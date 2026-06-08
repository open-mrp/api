package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to reopen a sales order.
type OpenSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Whether to notify the customer.
	NotifyCustomer bool `json:"notify_customer"`
}

var sampleOpenSalesOrderRequest = &OpenSalesOrderRequest{NotifyCustomer: false}

func (*OpenSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleOpenSalesOrderRequest)
}

// Reopens a sales order, transitioning it from fulfilled back to issued.
type OpenSalesOrderEndpoint struct{}

func (e *OpenSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*OpenSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*OpenSalesOrderRequest, *apiresource.SalesOrder]{
		Title:             "Reopen Sales Order",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/actions/open",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *OpenSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).OpenSalesOrder
		},
		// Status-action endpoints are commands: they return the updated sales
		// order without `?include=` expansion. Clients that need expanded
		// sub-resources re-fetch via GET /sales-orders/{id}?include=...
	})
}
