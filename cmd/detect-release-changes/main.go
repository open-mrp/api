package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/open-mrp/api/internal/releasechanges"
)

type outputs struct {
	PreviousTag  string
	CurrentRef   string
	ServicesJSON string
	ServicesCSV  string
	MatrixJSON   string
	HasServices  string
}

// errBadFlags signals a flag-parse failure; the FlagSet already printed the
// message and usage to stderr, so main exits nonzero without re-printing.
var errBadFlags = errors.New("invalid command-line flags")

func main() {
	ctx := context.Background()
	if err := Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, errBadFlags) {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(1)
	}
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	var currentTag string
	var repoRoot string

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&currentTag, "current-tag", "", "Current release tag, for example v0.18.3")
	flags.StringVar(&repoRoot, "repo-root", ".", "Repository root")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return errBadFlags
	}

	if currentTag == "" {
		return errors.New("missing required --current-tag")
	}

	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve repo root: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := gitFetchTags(ctx, absRepoRoot); err != nil {
		return fmt.Errorf("fetch git tags: %w", err)
	}

	previousTag, err := previousReleaseTag(ctx, absRepoRoot, currentTag)
	if err != nil {
		return fmt.Errorf("find previous release tag: %w", err)
	}

	currentRef, err := currentRefForTag(ctx, absRepoRoot, currentTag)
	if err != nil {
		return fmt.Errorf("resolve current ref: %w", err)
	}

	changedFiles, err := changedFilesBetween(ctx, absRepoRoot, previousTag, currentRef)
	if err != nil {
		return fmt.Errorf("list changed files: %w", err)
	}

	dirToServices, err := buildDependencyMap(ctx, absRepoRoot)
	if err != nil {
		return fmt.Errorf("build service dependency map: %w", err)
	}

	analysis := releasechanges.Analyze(changedFiles, dirToServices)

	out, err := marshalOutputs(previousTag, currentRef, analysis)
	if err != nil {
		return fmt.Errorf("marshal outputs: %w", err)
	}

	printSummary(stdout, previousTag, currentRef, changedFiles, analysis)
	if err := writeOutputs(stdout, getenv, out); err != nil {
		return fmt.Errorf("write outputs: %w", err)
	}

	return nil
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
	servicesJSON, err := json.Marshal(analysis.Services)
	if err != nil {
		return outputs{}, err
	}

	matrixJSON, err := json.Marshal(map[string][]string{
		"service": analysis.Services,
	})
	if err != nil {
		return outputs{}, err
	}

	return outputs{
		PreviousTag:  previousTag,
		CurrentRef:   currentRef,
		ServicesJSON: string(servicesJSON),
		ServicesCSV:  strings.Join(analysis.Services, ","),
		MatrixJSON:   string(matrixJSON),
		HasServices:  fmt.Sprintf("%t", len(analysis.Services) > 0),
	}, nil
}

func printSummary(stdout io.Writer, previousTag, currentRef string, changedFiles []string, analysis releasechanges.Analysis) {
	fmt.Fprintf(stdout, "Previous tag: %s\n", valueOrNone(previousTag))
	fmt.Fprintf(stdout, "Current ref: %s\n", currentRef)
	fmt.Fprintf(stdout, "Changed files: %d\n", len(changedFiles))
	fmt.Fprintf(stdout, "Services: %s\n", joinOrNone(analysis.Services))
}

func writeOutputs(stdout io.Writer, getenv func(string) string, out outputs) error {
	lines := []string{
		"previous_tag=" + out.PreviousTag,
		"current_ref=" + out.CurrentRef,
		"services_json=" + out.ServicesJSON,
		"services_csv=" + out.ServicesCSV,
		"matrix_json=" + out.MatrixJSON,
		"has_services=" + out.HasServices,
	}

	for _, line := range lines {
		fmt.Fprintln(stdout, line)
	}

	outputPath := getenv("GITHUB_OUTPUT")
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
