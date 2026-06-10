package httpgroup

import (
	"fmt"

	deliveryep "github.com/augno/api/services/api-gateway/endpoints/deliveries"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type DeliveriesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type DeliveriesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *DeliveriesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("deliveries endpoint group: core client is required")
	}
	return nil
}

func (*DeliveriesEndpointGroup) Materialize(config *DeliveriesEndpointGroupConfig) *DeliveriesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	deliverySvc := deliveryep.NewDeliverySvc(&deliveryep.DeliverySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Deliveries",
		Description:  "List and view deliveries.",
		ResourceType: &apiresource.Delivery{},
	}

	listDeliveriesEndpoint := apiendpoint.From(&deliveryep.ListDeliveriesEndpoint{}).WithService(inner, deliverySvc)
	getDeliveryEndpoint := apiendpoint.From(&deliveryep.RetrieveDeliveryEndpoint{}).WithService(inner, deliverySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listDeliveriesEndpoint,
		getDeliveryEndpoint,
	}

	return &DeliveriesEndpointGroup{inner}
}
