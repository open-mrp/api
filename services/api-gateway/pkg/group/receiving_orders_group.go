package httpgroup

import (
	"fmt"

	receivingorderep "github.com/augno/api/services/api-gateway/endpoints/receiving-orders"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ReceivingOrdersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ReceivingOrdersEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ReceivingOrdersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("receiving orders endpoint group: core client is required")
	}
	return nil
}

func (*ReceivingOrdersEndpointGroup) Materialize(config *ReceivingOrdersEndpointGroupConfig) *ReceivingOrdersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := receivingorderep.NewReceivingOrderSvc(&receivingorderep.ReceivingOrderSvcConfig{
		CoreClient: config.CoreClient.Receiving,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Receiving Orders",
		Description:  "List, view, stock, receive, void, and update receiving orders and receiving order lines.",
		ResourceType: &apiresource.ReceivingOrder{},
	}

	listEndpoint := apiendpoint.From(&receivingorderep.ListReceivingOrdersEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&receivingorderep.RetrieveReceivingOrderEndpoint{}).WithService(inner, svc)
	stockEndpoint := apiendpoint.From(&receivingorderep.StockReceivingOrderEndpoint{}).WithService(inner, svc)
	receiveEndpoint := apiendpoint.From(&receivingorderep.ReceiveReceivingOrderEndpoint{}).WithService(inner, svc)
	voidEndpoint := apiendpoint.From(&receivingorderep.VoidReceivingOrderEndpoint{}).WithService(inner, svc)
	updateLineEndpoint := apiendpoint.From(&receivingorderep.UpdateReceivingOrderLineEndpoint{}).WithService(inner, svc)
	voidLineEndpoint := apiendpoint.From(&receivingorderep.VoidReceivingOrderLineEndpoint{}).WithService(inner, svc)
	receiveLineEndpoint := apiendpoint.From(&receivingorderep.ReceiveReceivingOrderLineEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		stockEndpoint,
		receiveEndpoint,
		voidEndpoint,
		updateLineEndpoint,
		voidLineEndpoint,
		receiveLineEndpoint,
	}

	return &ReceivingOrdersEndpointGroup{inner}
}
