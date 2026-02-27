package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	httpgroup "github.com/augno/api/services/api-gateway/pkg/group"
	authpb "github.com/augno/api/shared/proto/auth"
	billingpb "github.com/augno/api/shared/proto/billing"
	pbgrpc "github.com/augno/api/shared/proto/core"
	platformpb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/version"
)

func main() {
	name := flag.String("name", "api", "Name of the API")
	rootDir := flag.String("root", "..", "Project root directory (paths are relative to this)")
	transformsPath := flag.String("transforms", "tools/apidocs/transforms.json", "Path to transforms JSON file")
	flag.Parse()

	if err := os.Chdir(*rootDir); err != nil {
		log.Fatalf("Error changing to root directory %s: %v", *rootDir, err)
	}

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

	coreClient := &grpcclient.CoreServiceClient{
		Client: struct{ pbgrpc.CoreServiceClient }{},
	}

	billingClient := &grpcclient.BillingServiceClient{
		Client: struct{ billingpb.BillingServiceClient }{},
	}

	platformClient := &grpcclient.PlatformServiceClient{
		LoggingClient: struct {
			platformpb.LoggingServiceClient
		}{},
	}

	groups := []apiendpoint.APIEndpointGroup{
		*(&httpgroup.HealthEndpointGroup{}).Materialize(httpgroup.HealthEndpointGroupConfig{}).APIEndpointGroup,
		*(&httpgroup.AuthEndpointGroup{}).Materialize(&httpgroup.AuthEndpointGroupConfig{
			AuthClient: authClient,
		}).APIEndpointGroup,
		*(&httpgroup.APIKeysEndpointGroup{}).Materialize(&httpgroup.APIKeysEndpointGroupConfig{
			AuthClient: authClient,
		}).APIEndpointGroup,
		*(&httpgroup.SandboxesEndpointGroup{}).Materialize(&httpgroup.SandboxesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.BillingEndpointGroup{}).Materialize(&httpgroup.BillingEndpointGroupConfig{
			BillingClient: billingClient,
		}).APIEndpointGroup,
		*(&httpgroup.RegistrationSessionsEndpointGroup{}).Materialize(&httpgroup.RegistrationSessionsEndpointGroupConfig{
			AuthClient: authClient,
		}).APIEndpointGroup,
		*(&httpgroup.RequestLogsEndpointGroup{}).Materialize(&httpgroup.RequestLogsEndpointGroupConfig{
			PlatformClient: platformClient,
		}).APIEndpointGroup,
		*(&httpgroup.UnitsEndpointGroup{}).Materialize(&httpgroup.UnitsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
	}

	generate(groups, "specs/public_openapi_spec.json", true, transforms, ver)
	generate(groups, "specs/internal_openapi_spec.json", false, transforms, ver)
}
