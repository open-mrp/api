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

// Request to close a sales order.
type CloseSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Whether to notify the customer.
	NotifyCustomer bool `json:"notify_customer"`
}

var sampleCloseSalesOrderRequest = &CloseSalesOrderRequest{NotifyCustomer: false}

func (*CloseSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCloseSalesOrderRequest)
}

// Closes a sales order, transitioning it from issued to fulfilled.
type CloseSalesOrderEndpoint struct{}

func (e *CloseSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CloseSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*CloseSalesOrderRequest, *apiresource.SalesOrder]{
		Title:             "Close Sales Order",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/actions/close",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CloseSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).CloseSalesOrder
		},
		// Status-action endpoints are commands: they return the updated sales
		// order without `?include=` expansion. Clients that need expanded
		// sub-resources re-fetch via GET /sales-orders/{id}?include=...
	})
}
