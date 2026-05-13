package httpgroup

import (
	"fmt"

	salesorderep "github.com/augno/api/services/api-gateway/endpoints/sales-orders"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SalesOrdersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SalesOrdersEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *SalesOrdersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sales orders endpoint group: core client is required")
	}
	return nil
}

func (*SalesOrdersEndpointGroup) Materialize(config *SalesOrdersEndpointGroupConfig) *SalesOrdersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := salesorderep.NewSalesOrderSvc(&salesorderep.SalesOrderSvcConfig{
		CoreClient: config.CoreClient.Sales,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Sales Orders",
		Description:  "List, view, create, update, and delete sales orders.",
		ResourceType: &apiresource.SalesOrderDetail{},
	}

	listEndpoint := (&salesorderep.ListSalesOrdersEndpoint{}).Materialize().WithService(inner, svc)
	retrieveEndpoint := (&salesorderep.RetrieveSalesOrderEndpoint{}).Materialize().WithService(inner, svc)
	createEndpoint := (&salesorderep.CreateSalesOrderEndpoint{}).Materialize().WithService(inner, svc)
	updateEndpoint := (&salesorderep.UpdateSalesOrderEndpoint{}).Materialize().WithService(inner, svc)
	deleteEndpoint := (&salesorderep.DeleteSalesOrderEndpoint{}).Materialize().WithService(inner, svc)
	bulkDeleteEndpoint := (&salesorderep.BulkDeleteSalesOrdersEndpoint{}).Materialize().WithService(inner, svc)
	changeStatusEndpoint := (&salesorderep.ChangeSalesOrderStatusEndpoint{}).Materialize().WithService(inner, svc)
	checkoutEndpoint := (&salesorderep.CheckoutSalesOrderEndpoint{}).Materialize().WithService(inner, svc)
	createProductionRunEndpoint := (&salesorderep.CreateProductionRunEndpoint{}).Materialize().WithService(inner, svc)
	createLineEndpoint := (&salesorderep.CreateSalesOrderLineEndpoint{}).Materialize().WithService(inner, svc)
	updateLineEndpoint := (&salesorderep.UpdateSalesOrderLineEndpoint{}).Materialize().WithService(inner, svc)
	deleteLineEndpoint := (&salesorderep.DeleteSalesOrderLineEndpoint{}).Materialize().WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		bulkDeleteEndpoint,
		changeStatusEndpoint,
		checkoutEndpoint,
		createProductionRunEndpoint,
		createLineEndpoint,
		updateLineEndpoint,
		deleteLineEndpoint,
	}

	return &SalesOrdersEndpointGroup{inner}
}
