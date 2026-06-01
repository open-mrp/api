package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/augno/api/internal/releasechanges"
)

type outputs struct {
	PreviousTag        string
	CurrentRef         string
	BuildServicesJSON  string
	BuildServicesCSV   string
	BuildMatrixJSON    string
	HasBuildServices   string
	DeployServicesJSON string
	DeployServicesCSV  string
	HasDeployServices  string
	TerraformChanged   string
	ConfigChanged      string
	PlatformChanged    string
}

func main() {
	var currentTag string
	var repoRoot string

	flag.StringVar(&currentTag, "current-tag", "", "Current release tag, for example v0.18.3")
	flag.StringVar(&repoRoot, "repo-root", ".", "Repository root")
	flag.Parse()

	if currentTag == "" {
		fail("missing required --current-tag")
	}

	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		fail("resolve repo root: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := gitFetchTags(ctx, absRepoRoot); err != nil {
		fail("fetch git tags: %v", err)
	}

	previousTag, err := previousReleaseTag(ctx, absRepoRoot, currentTag)
	if err != nil {
		fail("find previous release tag: %v", err)
	}

	currentRef, err := currentRefForTag(ctx, absRepoRoot, currentTag)
	if err != nil {
		fail("resolve current ref: %v", err)
	}

	changedFiles, err := changedFilesBetween(ctx, absRepoRoot, previousTag, currentRef)
	if err != nil {
		fail("list changed files: %v", err)
	}

	dirToServices, err := buildDependencyMap(ctx, absRepoRoot)
	if err != nil {
		fail("build service dependency map: %v", err)
	}

	analysis := releasechanges.Analyze(changedFiles, dirToServices)

	out, err := marshalOutputs(previousTag, currentRef, analysis)
	if err != nil {
		fail("marshal outputs: %v", err)
	}

	printSummary(previousTag, currentRef, changedFiles, analysis)
	if err := writeOutputs(out); err != nil {
		fail("write outputs: %v", err)
	}
}

func gitFetchTags(ctx context.Context, repoRoot string) error {
	cmd := exec.CommandContext(ctx, "git", "fetch", "--tags", "--force", "origin")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func previousReleaseTag(ctx context.Context, repoRoot, currentTag string) (string, error) {
	output, err := gitOutput(ctx, repoRoot, "tag", "-l", "v*", "--sort=-version:refname")
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		tag := strings.TrimSpace(scanner.Text())
		if tag == "" || tag == currentTag {
			continue
		}
		return tag, nil
	}

	return "", scanner.Err()
}

func currentRefForTag(ctx context.Context, repoRoot, currentTag string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "-q", currentTag) // #nosec G204 -- args are controlled
	cmd.Dir = repoRoot
	if err := cmd.Run(); err == nil {
		return currentTag, nil
	}

	return "HEAD", nil
}

func changedFilesBetween(ctx context.Context, repoRoot, previousTag, currentRef string) ([]string, error) {
	if previousTag == "" {
		output, err := gitOutput(ctx, repoRoot, "ls-files")
		if err != nil {
			return nil, err
		}
		return nonEmptyLines(output), nil
	}

	output, err := gitOutput(ctx, repoRoot, "diff", "--name-only", previousTag, currentRef)
	if err != nil {
		return nil, err
	}

	return nonEmptyLines(output), nil
}

func buildDependencyMap(ctx context.Context, repoRoot string) (map[string][]string, error) {
	dirToServiceSet := make(map[string]map[string]struct{})

	for _, service := range releasechanges.ServiceNames {
		args := []string{"list", "-deps", "-f", "{{.Dir}}", "./services/" + service + "/cmd/..."}
		cmd := exec.CommandContext(ctx, "go", args...) // #nosec G204 -- args are controlled
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")

		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}

		for _, dir := range nonEmptyLines(string(output)) {
			relDir, ok := repoRelativeDir(repoRoot, dir)
			if !ok {
				continue
			}
			if _, ok := dirToServiceSet[relDir]; !ok {
				dirToServiceSet[relDir] = make(map[string]struct{}, 1)
			}
			dirToServiceSet[relDir][service] = struct{}{}
		}
	}

	dirToServices := make(map[string][]string, len(dirToServiceSet))
	for dir, serviceSet := range dirToServiceSet {
		services := make([]string, 0, len(serviceSet))
		for _, service := range releasechanges.ServiceNames {
			if _, ok := serviceSet[service]; ok {
				services = append(services, service)
			}
		}
		dirToServices[dir] = services
	}

	return dirToServices, nil
}

func repoRelativeDir(repoRoot, dir string) (string, bool) {
	if dir == "" {
		return "", false
	}

	relDir, err := filepath.Rel(repoRoot, dir)
	if err != nil {
		return "", false
	}

	relDir = filepath.ToSlash(relDir)
	if relDir == "." || strings.HasPrefix(relDir, "../") {
		return "", false
	}

	return relDir, true
}

func marshalOutputs(previousTag, currentRef string, analysis releasechanges.Analysis) (outputs, error) {
	buildJSON, err := json.Marshal(analysis.BuildServices)
	if err != nil {
		return outputs{}, err
	}

	buildMatrixJSON, err := json.Marshal(map[string][]string{
		"service": analysis.BuildServices,
	})
	if err != nil {
		return outputs{}, err
	}

	deployJSON, err := json.Marshal(analysis.DeployServices)
	if err != nil {
		return outputs{}, err
	}

	return outputs{
		PreviousTag:        previousTag,
		CurrentRef:         currentRef,
		BuildServicesJSON:  string(buildJSON),
		BuildServicesCSV:   strings.Join(analysis.BuildServices, ","),
		BuildMatrixJSON:    string(buildMatrixJSON),
		HasBuildServices:   fmt.Sprintf("%t", len(analysis.BuildServices) > 0),
		DeployServicesJSON: string(deployJSON),
		DeployServicesCSV:  strings.Join(analysis.DeployServices, ","),
		HasDeployServices:  fmt.Sprintf("%t", len(analysis.DeployServices) > 0),
		TerraformChanged:   fmt.Sprintf("%t", analysis.TerraformChanged),
		ConfigChanged:      fmt.Sprintf("%t", analysis.ConfigChanged),
		PlatformChanged:    fmt.Sprintf("%t", analysis.PlatformChanged),
	}, nil
}

func printSummary(previousTag, currentRef string, changedFiles []string, analysis releasechanges.Analysis) {
	fmt.Printf("Previous tag: %s\n", valueOrNone(previousTag))
	fmt.Printf("Current ref: %s\n", currentRef)
	fmt.Printf("Changed files: %d\n", len(changedFiles))
	fmt.Printf("Build services: %s\n", joinOrNone(analysis.BuildServices))
	fmt.Printf("Deploy services: %s\n", joinOrNone(analysis.DeployServices))
	fmt.Printf("Terraform changed: %t\n", analysis.TerraformChanged)
	fmt.Printf("Config changed: %t\n", analysis.ConfigChanged)
	fmt.Printf("Platform changed: %t\n", analysis.PlatformChanged)
}

func writeOutputs(out outputs) error {
	lines := []string{
		"previous_tag=" + out.PreviousTag,
		"current_ref=" + out.CurrentRef,
		"build_services_json=" + out.BuildServicesJSON,
		"build_services_csv=" + out.BuildServicesCSV,
		"build_matrix_json=" + out.BuildMatrixJSON,
		"has_build_services=" + out.HasBuildServices,
		"deploy_services_json=" + out.DeployServicesJSON,
		"deploy_services_csv=" + out.DeployServicesCSV,
		"has_deploy_services=" + out.HasDeployServices,
		"terraform_changed=" + out.TerraformChanged,
		"config_changed=" + out.ConfigChanged,
		"platform_changed=" + out.PlatformChanged,
	}

	for _, line := range lines {
		fmt.Println(line)
	}

	outputPath := os.Getenv("GITHUB_OUTPUT")
	if outputPath == "" {
		return nil
	}

	var buffer bytes.Buffer
	for _, line := range lines {
		buffer.WriteString(line)
		buffer.WriteByte('\n')
	}

	cleanPath := filepath.Clean(outputPath)
	return os.WriteFile(cleanPath, buffer.Bytes(), 0o600) // #nosec G306,G703 -- CI output file, path from GITHUB_OUTPUT env var
}

func gitOutput(ctx context.Context, repoRoot string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...) // #nosec G204 -- args are controlled
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func nonEmptyLines(output string) []string {
	lines := strings.Split(output, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func valueOrNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(none)"
	}
	return value
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
