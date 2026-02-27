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
		Title:        "Sandbox Management",
		Description:  "Handles listing and managing sandbox environments.",
		ResourceType: &apiresource.Sandbox{},
	}

	listSandboxesEndpoint := (&sandboxep.ListSandboxesEndpoint{}).Materialize().WithService(inner, sandboxSvc)
	getSandboxEndpoint := (&sandboxep.GetSandboxEndpoint{}).Materialize().WithService(inner, sandboxSvc)
	createSandboxEndpoint := (&sandboxep.CreateSandboxEndpoint{}).Materialize().WithService(inner, sandboxSvc)
	deleteSandboxEndpoint := (&sandboxep.DeleteSandboxEndpoint{}).Materialize().WithService(inner, sandboxSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listSandboxesEndpoint,
		getSandboxEndpoint,
		createSandboxEndpoint,
		deleteSandboxEndpoint,
	}

	return &SandboxesEndpointGroup{inner}
}
