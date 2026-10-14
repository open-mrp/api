package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/open-mrp/apikit/openapi"

	"github.com/open-mrp/apikit/version"
)

var quietOutput bool

// errBadFlags signals a flag-parse failure; the FlagSet already printed the
// message and usage to stderr, so main exits nonzero without re-printing.
var errBadFlags = errors.New("invalid command-line flags")

func logInfof(format string, args ...any) {
	if quietOutput {
		return
	}
	log.Printf(format, args...)
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "api", "Name of the API")
	rootDir := flags.String("root", "..", "Project root directory (paths are relative to this)")
	transformsPath := flags.String("transforms", "tools/apidocs/transforms.json", "Path to transforms JSON file")
	httpieMode := flags.Bool("httpie", false, "Generate HTTPie workspace file instead of OpenAPI specs")
	agentToolsMode := flags.Bool("agent-tools", false, "Generate the agent-service endpoint-tool catalog + seed instead of OpenAPI specs")
	withAgentTools := flags.Bool("with-agent-tools", false, "Also generate the agent-service endpoint-tool catalog (after OpenAPI specs and Stainless configs)")
	skipStainless := flags.Bool("skip-stainless", false, "Skip Stainless config generation (only generate OpenAPI specs)")
	onlyStainless := flags.Bool("only-stainless", false, "Only generate Stainless configs (skip OpenAPI specs)")
	quiet := flags.Bool("quiet", false, "Suppress informational output")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return errBadFlags
	}

	if *skipStainless && *onlyStainless {
		return errors.New("--skip-stainless and --only-stainless are mutually exclusive")
	}
	quietOutput = *quiet
	log.SetOutput(stdout)

	if err := os.Chdir(*rootDir); err != nil {
		return fmt.Errorf("changing to root directory %s: %w", *rootDir, err)
	}

	ver := version.Latest().Version
	logInfof("Generating OpenAPI spec for %s (version %s)...", *name, ver)

	var transforms []openapi.Transform
	if *transformsPath != "" {
		if _, err := os.Stat(*transformsPath); err == nil {
			b, err := os.ReadFile(*transformsPath)
			if err != nil {
				log.Printf("Warning: error reading transforms file: %v", err)
			} else {
				if err := json.Unmarshal(b, &transforms); err != nil {
					log.Printf("Warning: error unmarshaling transforms: %v", err)
				} else {
					logInfof("Loaded %d transforms from %s", len(transforms), *transformsPath)
				}
			}
		}
	}

	groups := openAPIEndpointGroups()

	if *httpieMode {
		return generateHTTPieWorkspace(groups, "httpie/httpie-space.json")
	}

	public := specConfig(groups, true, transforms, ver)
	internal := specConfig(groups, false, transforms, ver)

	if *agentToolsMode {
		return generateAgentTools(internal)
	}

	if !*onlyStainless {
		start := time.Now()
		for path, cfg := range map[string]openapi.Config{
			"specs/public_openapi_spec.json":   public,
			"specs/internal_openapi_spec.json": internal,
		} {
			if err := openapi.WriteFile(cfg, path); err != nil {
				return fmt.Errorf("generating spec %s: %w", path, err)
			}
		}
		logInfof("OpenAPI spec generation completed in %s", time.Since(start).Round(time.Millisecond))
	}

	if !*skipStainless {
		for path, cfg := range map[string]openapi.Config{
			"stainless/public/stainless.yml":   public,
			"stainless/internal/stainless.yml": internal,
		} {
			if err := openapi.WriteStainlessConfig(cfg, path); err != nil {
				return fmt.Errorf("generating Stainless config %s: %w", path, err)
			}
		}
	}

	if *withAgentTools {
		if err := generateAgentTools(internal); err != nil {
			return err
		}
	}

	return nil
}
