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

	listEndpoint := (&receivingorderep.ListReceivingOrdersEndpoint{}).Materialize().WithService(inner, svc)
	retrieveEndpoint := (&receivingorderep.RetrieveReceivingOrderEndpoint{}).Materialize().WithService(inner, svc)
	stockEndpoint := (&receivingorderep.StockReceivingOrderEndpoint{}).Materialize().WithService(inner, svc)
	receiveEndpoint := (&receivingorderep.ReceiveReceivingOrderEndpoint{}).Materialize().WithService(inner, svc)
	voidEndpoint := (&receivingorderep.VoidReceivingOrderEndpoint{}).Materialize().WithService(inner, svc)
	updateLineEndpoint := (&receivingorderep.UpdateReceivingOrderLineEndpoint{}).Materialize().WithService(inner, svc)
	voidLineEndpoint := (&receivingorderep.VoidReceivingOrderLineEndpoint{}).Materialize().WithService(inner, svc)
	receiveLineEndpoint := (&receivingorderep.ReceiveReceivingOrderLineEndpoint{}).Materialize().WithService(inner, svc)

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
