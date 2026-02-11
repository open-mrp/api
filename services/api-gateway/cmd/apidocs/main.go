package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	httpgroup "github.com/augno/api/services/api-gateway/internal/router/group"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	authpb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/version"
)

func main() {
	name := flag.String("name", "api", "Name of the API")
	transformsPath := flag.String("transforms", "services/api-gateway/cmd/apidocs/transforms.json", "Path to transforms JSON file")
	flag.Parse()

	ver := version.Latest.Version
	log.Printf("Generating OpenAPI spec for %s (version %s)...", *name, ver)

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

	// Initialize endpoint groups with "dummy" clients using anonymous structs
	authClient := &grpcclient.AuthServiceClient{
		Client: struct{ authpb.AuthServiceClient }{},
	}

	groups := []apiendpoint.APIEndpointGroup{
		*(&httpgroup.HealthEndpointGroup{}).Materialize(httpgroup.HealthEndpointGroupConfig{}).APIEndpointGroup,
		*(&httpgroup.AuthEndpointGroup{}).Materialize(httpgroup.AuthEndpointGroupConfig{
			AuthClient: authClient,
		}).APIEndpointGroup,
	}

	generate(groups, "specs/public_openapi_spec.json", true, transforms, ver)
	generate(groups, "specs/internal_openapi_spec.json", false, transforms, ver)
}
