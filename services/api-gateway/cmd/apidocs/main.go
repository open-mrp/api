package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	httpgroup "github.com/augno/api/services/api-gateway/internal/router/group"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/auth"
)

func main() {
	name := flag.String("name", "api", "Name of the API")
	transformsPath := flag.String("transforms", "services/api-gateway/cmd/apidocs/transforms.json", "Path to transforms JSON file")
	manifestPath := flag.String("manifest", ".release-please-manifest.json", "Path to .release-please-manifest.json")
	flag.Parse()

	log.Printf("Generating OpenAPI spec for %s...", *name)

	version := "1.0.0"
	if *manifestPath != "" {
		if b, err := os.ReadFile(*manifestPath); err == nil {
			var manifest map[string]string
			if err := json.Unmarshal(b, &manifest); err == nil {
				if v, ok := manifest["."]; ok {
					version = v
					log.Printf("Detected version %s from manifest", version)
				}
			}
		}
	}

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
					log.Printf("Loaded %d transforms from %s", len(transforms), *transformsPath)
				}
			}
		}
	}

	// Initialize endpoint groups with a "dummy" client using an anonymous struct
	authClient := &grpcclient.AuthServiceClient{
		Client: struct{ pb.AuthServiceClient }{},
	}

	groups := []apiendpoint.APIEndpointGroup{
		*(&httpgroup.HealthEndpointGroup{}).Materialize(httpgroup.HealthEndpointGroupConfig{}).APIEndpointGroup,
		*(&httpgroup.AuthEndpointGroup{}).Materialize(httpgroup.AuthEndpointGroupConfig{
			PlatformMode: constants.PlatformModeDevelopment,
			AuthClient:   authClient,
		}).APIEndpointGroup,
	}

	generate(groups, "specs/public_openapi_spec.json", true, transforms, version)
	generate(groups, "specs/internal_openapi_spec.json", false, transforms, version)
}
