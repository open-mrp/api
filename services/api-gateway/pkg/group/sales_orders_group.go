package httpgroup

import (
	"fmt"

	salesorderep "github.com/open-mrp/api/services/api-gateway/endpoints/sales-orders"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
	cancelEndpoint := apiendpoint.From(&salesorderep.CancelSalesOrderEndpoint{}).WithService(inner, svc)
	preview9IssueEndpoint := apiendpoint.From(&salesorderep.Preview9IssueSalesOrderEndpoint{}).WithService(inner, svc)
	preview9UnissueEndpoint := apiendpoint.From(&salesorderep.Preview9UnissueSalesOrderEndpoint{}).WithService(inner, svc)
	preview9CloseEndpoint := apiendpoint.From(&salesorderep.Preview9CloseSalesOrderEndpoint{}).WithService(inner, svc)
	preview9OpenEndpoint := apiendpoint.From(&salesorderep.Preview9OpenSalesOrderEndpoint{}).WithService(inner, svc)
	checkoutEndpoint := apiendpoint.From(&salesorderep.CheckoutSalesOrderEndpoint{}).WithService(inner, svc)
	quotePricesEndpoint := apiendpoint.From(&salesorderep.QuoteSalesOrderPricesEndpoint{}).WithService(inner, svc)
	quoteFreightEndpoint := apiendpoint.From(&salesorderep.QuoteSalesOrderFreightEndpoint{}).WithService(inner, svc)
	quoteCommitmentEndpoint := apiendpoint.From(&salesorderep.QuoteSalesOrderCommitmentEndpoint{}).WithService(inner, svc)
	createProductionRunEndpoint := apiendpoint.From(&salesorderep.CreateProductionRunEndpoint{}).WithService(inner, svc)
	createLineEndpoint := apiendpoint.From(&salesorderep.CreateSalesOrderLineEndpoint{}).WithService(inner, svc)
	updateLineEndpoint := apiendpoint.From(&salesorderep.UpdateSalesOrderLineEndpoint{}).WithService(inner, svc)
	deleteLineEndpoint := apiendpoint.From(&salesorderep.DeleteSalesOrderLineEndpoint{}).WithService(inner, svc)
	reorderLinesEndpoint := apiendpoint.From(&salesorderep.ReorderSalesOrderLinesEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		bulkDeleteEndpoint,
		issueEndpoint,
		unissueEndpoint,
		cancelEndpoint,
		preview9IssueEndpoint,
		preview9UnissueEndpoint,
		preview9CloseEndpoint,
		preview9OpenEndpoint,
		checkoutEndpoint,
		quotePricesEndpoint,
		quoteFreightEndpoint,
		quoteCommitmentEndpoint,
		createProductionRunEndpoint,
		createLineEndpoint,
		updateLineEndpoint,
		deleteLineEndpoint,
		reorderLinesEndpoint,
	}

	return &SalesOrdersEndpointGroup{inner}
}
