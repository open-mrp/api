package httpgroup

import (
	"fmt"

	orderdiscountep "github.com/augno/api/services/api-gateway/endpoints/order-discounts"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type OrderDiscountsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type OrderDiscountsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *OrderDiscountsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("order discounts endpoint group: core client is required")
	}
	return nil
}

func (*OrderDiscountsEndpointGroup) Materialize(config *OrderDiscountsEndpointGroupConfig) *OrderDiscountsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := orderdiscountep.NewOrderDiscountSvc(&orderdiscountep.OrderDiscountSvcConfig{
		CoreClient: config.CoreClient.Sales,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Order Discounts",
		Description:  "List and manage order discounts.",
		ResourceType: &apiresource.OrderDiscount{},
	}

	listEndpoint := (&orderdiscountep.ListOrderDiscountsEndpoint{}).Materialize().WithService(inner, svc)
	retrieveEndpoint := (&orderdiscountep.RetrieveOrderDiscountEndpoint{}).Materialize().WithService(inner, svc)
	createEndpoint := (&orderdiscountep.CreateOrderDiscountEndpoint{}).Materialize().WithService(inner, svc)
	updateEndpoint := (&orderdiscountep.UpdateOrderDiscountEndpoint{}).Materialize().WithService(inner, svc)
	deleteEndpoint := (&orderdiscountep.DeleteOrderDiscountEndpoint{}).Materialize().WithService(inner, svc)
	findByCodeEndpoint := (&orderdiscountep.FindOrderDiscountByCodeEndpoint{}).Materialize().WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		findByCodeEndpoint,
	}

	return &OrderDiscountsEndpointGroup{inner}
}
