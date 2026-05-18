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

	listEndpoint := apiendpoint.From(&orderdiscountep.ListOrderDiscountsEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&orderdiscountep.RetrieveOrderDiscountEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&orderdiscountep.CreateOrderDiscountEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&orderdiscountep.UpdateOrderDiscountEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&orderdiscountep.DeleteOrderDiscountEndpoint{}).WithService(inner, svc)
	findByCodeEndpoint := apiendpoint.From(&orderdiscountep.FindOrderDiscountByCodeEndpoint{}).WithService(inner, svc)

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
