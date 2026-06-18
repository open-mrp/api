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
	// CoreClient (required) is the core-service gRPC client.
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
		ResourceType: &apiresource.SalesOrder{},
	}

	listEndpoint := apiendpoint.From(&salesorderep.ListSalesOrdersEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&salesorderep.RetrieveSalesOrderEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&salesorderep.CreateSalesOrderEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&salesorderep.UpdateSalesOrderEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&salesorderep.DeleteSalesOrderEndpoint{}).WithService(inner, svc)
	bulkDeleteEndpoint := apiendpoint.From(&salesorderep.BulkDeleteSalesOrdersEndpoint{}).WithService(inner, svc)
	issueEndpoint := apiendpoint.From(&salesorderep.IssueSalesOrderEndpoint{}).WithService(inner, svc)
	unissueEndpoint := apiendpoint.From(&salesorderep.UnissueSalesOrderEndpoint{}).WithService(inner, svc)
	closeEndpoint := apiendpoint.From(&salesorderep.CloseSalesOrderEndpoint{}).WithService(inner, svc)
	openEndpoint := apiendpoint.From(&salesorderep.OpenSalesOrderEndpoint{}).WithService(inner, svc)
	checkoutEndpoint := apiendpoint.From(&salesorderep.CheckoutSalesOrderEndpoint{}).WithService(inner, svc)
	quotePricesEndpoint := apiendpoint.From(&salesorderep.QuoteSalesOrderPricesEndpoint{}).WithService(inner, svc)
	createProductionRunEndpoint := apiendpoint.From(&salesorderep.CreateProductionRunEndpoint{}).WithService(inner, svc)
	createLineEndpoint := apiendpoint.From(&salesorderep.CreateSalesOrderLineEndpoint{}).WithService(inner, svc)
	updateLineEndpoint := apiendpoint.From(&salesorderep.UpdateSalesOrderLineEndpoint{}).WithService(inner, svc)
	deleteLineEndpoint := apiendpoint.From(&salesorderep.DeleteSalesOrderLineEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		bulkDeleteEndpoint,
		issueEndpoint,
		unissueEndpoint,
		closeEndpoint,
		openEndpoint,
		checkoutEndpoint,
		quotePricesEndpoint,
		createProductionRunEndpoint,
		createLineEndpoint,
		updateLineEndpoint,
		deleteLineEndpoint,
	}

	return &SalesOrdersEndpointGroup{inner}
}
