package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// errBadFlags signals a flag-parse failure; the FlagSet already printed the
// message and usage to stderr, so main exits nonzero without re-printing.
var errBadFlags = errors.New("invalid command-line flags")

var releaseTagPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z][0-9A-Za-z.-]*)?$`)

// tagLookup reports the image tags an ECR repository holds.
type tagLookup func(ctx context.Context, repository string) (map[string]struct{}, error)

// group is one dispatch: the services that roll to the same image tag.
type group struct {
	Tag      string
	Services []string
}

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
	var toTag string
	var services string
	var repoRoot string
	var repositoryPrefix string

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&toTag, "to-tag", "", "Release tag being rolled back to, for example v2.7.4")
	flags.StringVar(&services, "services", "", "Comma-separated services to resolve an image tag for")
	flags.StringVar(&repoRoot, "repo-root", ".", "Repository root")
	flags.StringVar(&repositoryPrefix, "repository-prefix", "augno",
		"ECR repository prefix; a service's images live at <prefix>/<service>")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return errBadFlags
	}

	if toTag == "" {
		return errors.New("missing required --to-tag")
	}

	serviceNames := splitList(services)
	if len(serviceNames) == 0 {
		return errors.New("missing required --services")
	}

	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve repo root: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	candidates, err := candidateTags(ctx, absRepoRoot, toTag)
	if err != nil {
		return fmt.Errorf("list candidate release tags: %w", err)
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no release tags at or before %s", toTag)
	}

	resolved, unresolved, err := resolveImageTags(ctx, repositoryPrefix, serviceNames, candidates, ecrTags)
	if err != nil {
		return err
	}
	if len(unresolved) > 0 {
		return fmt.Errorf("no image at or before %s for: %s", toTag, strings.Join(unresolved, ", "))
	}

	groups := groupByTag(candidates, serviceNames, resolved)
	printSummary(stdout, toTag, candidates, serviceNames, resolved)
	for _, g := range groups {
		fmt.Fprintf(stdout, "group=%s %s\n", g.Tag, strings.Join(g.Services, ","))
	}

	return nil
}

// candidateTags lists the release tags reachable from toTag, newest first.
func candidateTags(ctx context.Context, repoRoot, toTag string) ([]string, error) {
	output, err := gitOutput(ctx, repoRoot, "tag", "--list", "v*", "--sort=-v:refname", "--merged", toTag)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, line := range orderedLines(output) {
		if releaseTagPattern.MatchString(line) {
			tags = append(tags, line)
		}
	}

	return tags, nil
}

// resolveImageTags picks, per service, the newest candidate tag ECR actually holds an image for.
//
// A release only builds the services that changed in it, so most release tags have no image for
// most services. Falling back to an older tag is not rolling that service back further than asked:
// no image means the service did not change, so the older image is the same code.
func resolveImageTags(
	ctx context.Context,
	repositoryPrefix string,
	serviceNames, candidates []string,
	lookup tagLookup,
) (map[string]string, []string, error) {
	resolved := make(map[string]string, len(serviceNames))
	var unresolved []string

	for _, service := range serviceNames {
		tags, err := lookup(ctx, repositoryPrefix+"/"+service)
		if err != nil {
			return nil, nil, err
		}

		matched := ""
		for _, candidate := range candidates {
			if _, ok := tags[candidate]; ok {
				matched = candidate
				break
			}
		}

		if matched == "" {
			unresolved = append(unresolved, service)
			continue
		}
		resolved[service] = matched
	}

	return resolved, unresolved, nil
}

func groupByTag(candidates, serviceNames []string, resolved map[string]string) []group {
	var groups []group

	for _, candidate := range candidates {
		var members []string
		for _, service := range serviceNames {
			if resolved[service] == candidate {
				members = append(members, service)
			}
		}
		if len(members) > 0 {
			groups = append(groups, group{Tag: candidate, Services: members})
		}
	}

	return groups
}

func ecrTags(ctx context.Context, repository string) (map[string]struct{}, error) {
	cmd := exec.CommandContext(ctx, "aws", "ecr", "list-images", // #nosec G204 -- args are controlled
		"--repository-name", repository,
		"--filter", "tagStatus=TAGGED",
		"--query", "imageIds[].imageTag",
		"--output", "text")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "RepositoryNotFoundException") {
			return nil, nil
		}
		return nil, fmt.Errorf("list images in %s: %w: %s", repository, err, strings.TrimSpace(string(output)))
	}

	tags := make(map[string]struct{})
	for field := range strings.FieldsSeq(string(output)) {
		tags[field] = struct{}{}
	}

	return tags, nil
}

func printSummary(stdout io.Writer, toTag string, candidates, serviceNames []string, resolved map[string]string) {
	fmt.Fprintf(stdout, "Rolling back to: %s\n", toTag)
	fmt.Fprintf(stdout, "Candidate tags: %d\n", len(candidates))
	for _, service := range serviceNames {
		tag := resolved[service]
		if tag == toTag {
			fmt.Fprintf(stdout, "  %s -> %s\n", service, tag)
			continue
		}
		fmt.Fprintf(stdout, "  %s -> %s (newest image at or before %s)\n", service, tag, toTag)
	}
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

func orderedLines(output string) []string {
	lines := strings.Split(output, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}
