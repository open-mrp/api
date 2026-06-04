package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/augno/api/shared/version"
)

var quietOutput bool

func logInfof(format string, args ...any) {
	if quietOutput {
		return
	}
	log.Printf(format, args...)
}

func main() {
	name := flag.String("name", "api", "Name of the API")
	rootDir := flag.String("root", "..", "Project root directory (paths are relative to this)")
	transformsPath := flag.String("transforms", "tools/apidocs/transforms.json", "Path to transforms JSON file")
	httpieMode := flag.Bool("httpie", false, "Generate HTTPie workspace file instead of OpenAPI specs")
	skipStainless := flag.Bool("skip-stainless", false, "Skip Stainless config generation (only generate OpenAPI specs)")
	onlyStainless := flag.Bool("only-stainless", false, "Only generate Stainless configs (skip OpenAPI specs)")
	quiet := flag.Bool("quiet", false, "Suppress informational output")
	flag.Parse()

	if *skipStainless && *onlyStainless {
		log.Fatal("Error: --skip-stainless and --only-stainless are mutually exclusive")
	}
	quietOutput = *quiet

	if err := os.Chdir(*rootDir); err != nil {
		log.Fatalf("Error changing to root directory %s: %v", *rootDir, err)
	}

	ver := version.Latest.Version
	logInfof("Generating OpenAPI spec for %s (version %s)...", *name, ver)

	var transforms []Transform
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
		generateHTTPieWorkspace(groups, "httpie/httpie-space.json")
		return
	}

	if !*onlyStainless {
		start := time.Now()
		generate(groups, "specs/public_openapi_spec.json", true, transforms, ver)
		generate(groups, "specs/internal_openapi_spec.json", false, transforms, ver)
		log.Printf("OpenAPI spec generation completed in %s", time.Since(start).Round(time.Millisecond))
	}

	if !*skipStainless {
		generateStainlessConfigs(groups, ver)
	}
}
