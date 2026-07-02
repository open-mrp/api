package httpgroup

import (
	"fmt"

	quantityep "github.com/augno/api/services/api-gateway/endpoints/quantities"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type QuantitiesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type QuantitiesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *QuantitiesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("quantities endpoint group: core client is required")
	}
	return nil
}

func (*QuantitiesEndpointGroup) Materialize(config *QuantitiesEndpointGroupConfig) *QuantitiesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	quantitySvc := quantityep.NewQuantitySvc(&quantityep.QuantitySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Quantities",
		Description:  "Manage quantity records.",
		ResourceType: &apiresource.Quantity{},
	}

	updateQuantityEndpoint := apiendpoint.From(&quantityep.UpdateQuantityEndpoint{}).WithService(inner, quantitySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		updateQuantityEndpoint,
	}

	return &QuantitiesEndpointGroup{inner}
}
