package httpgroup

import (
	"fmt"

	priorityep "github.com/augno/api/services/api-gateway/endpoints/priorities"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type PrioritiesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PrioritiesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PrioritiesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("priorities endpoint group: core client is required")
	}
	return nil
}

func (*PrioritiesEndpointGroup) Materialize(config *PrioritiesEndpointGroupConfig) *PrioritiesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	prioritySvc := priorityep.NewPrioritySvc(&priorityep.PrioritySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Priorities",
		Description:  "List and retrieve priorities.",
		ResourceType: &apiresource.Priority{},
	}

	listPrioritiesEndpoint := apiendpoint.From(&priorityep.ListPrioritiesEndpoint{}).WithService(inner, prioritySvc)
	getPriorityEndpoint := apiendpoint.From(&priorityep.RetrievePriorityEndpoint{}).WithService(inner, prioritySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listPrioritiesEndpoint,
		getPriorityEndpoint,
	}

	return &PrioritiesEndpointGroup{inner}
}
