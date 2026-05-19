package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

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
	quiet := flag.Bool("quiet", false, "Suppress informational output")
	flag.Parse()
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

	generate(groups, "specs/public_openapi_spec.json", true, transforms, ver)
	generate(groups, "specs/internal_openapi_spec.json", false, transforms, ver)
}
