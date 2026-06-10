package httpgroup

import (
	"fmt"

	edirunep "github.com/augno/api/services/api-gateway/endpoints/edi-runs"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type EDIRunsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type EDIRunsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *EDIRunsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("edi runs endpoint group: core client is required")
	}
	return nil
}

func (*EDIRunsEndpointGroup) Materialize(config *EDIRunsEndpointGroupConfig) *EDIRunsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := edirunep.NewEDIRunSvc(&edirunep.EDIRunSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "EDI Runs",
		Description:  "List and view EDI runs.",
		ResourceType: &apiresource.EDIRun{},
	}

	listEndpoint := apiendpoint.From(&edirunep.ListEDIRunsEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&edirunep.RetrieveEDIRunEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
	}

	return &EDIRunsEndpointGroup{inner}
}
