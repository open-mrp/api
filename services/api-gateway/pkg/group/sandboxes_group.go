package httpgroup

import (
	"fmt"

	sandboxep "github.com/augno/api/services/api-gateway/endpoints/sandboxes"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SandboxesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SandboxesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *SandboxesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sandboxes endpoint group: core client is required")
	}
	return nil
}

func (*SandboxesEndpointGroup) Materialize(config *SandboxesEndpointGroupConfig) *SandboxesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	sandboxSvc := sandboxep.NewSandboxSvc(&sandboxep.SandboxSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Sandbox",
		Description:  "List and manage sandbox environments.",
		ResourceType: &apiresource.Sandbox{},
	}

	listSandboxesEndpoint := apiendpoint.From(&sandboxep.ListSandboxesEndpoint{}).WithService(inner, sandboxSvc)
	getSandboxEndpoint := apiendpoint.From(&sandboxep.RetrieveSandboxEndpoint{}).WithService(inner, sandboxSvc)
	createSandboxEndpoint := apiendpoint.From(&sandboxep.CreateSandboxEndpoint{}).WithService(inner, sandboxSvc)
	deleteSandboxEndpoint := apiendpoint.From(&sandboxep.DeleteSandboxEndpoint{}).WithService(inner, sandboxSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listSandboxesEndpoint,
		getSandboxEndpoint,
		createSandboxEndpoint,
		deleteSandboxEndpoint,
	}

	return &SandboxesEndpointGroup{inner}
}
