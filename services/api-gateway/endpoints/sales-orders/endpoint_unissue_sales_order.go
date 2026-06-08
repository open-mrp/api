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

// Request to unissue a sales order.
type UnissueSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Whether to notify the customer.
	NotifyCustomer bool `json:"notify_customer"`
}

var sampleUnissueSalesOrderRequest = &UnissueSalesOrderRequest{NotifyCustomer: false}

func (*UnissueSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUnissueSalesOrderRequest)
}

// Unissues a sales order, transitioning it from issued back to estimate.
type UnissueSalesOrderEndpoint struct{}

func (e *UnissueSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnissueSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*UnissueSalesOrderRequest, *apiresource.SalesOrder]{
		Title:             "Unissue Sales Order",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/actions/unissue",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).UnissueSalesOrder
		},
		// Status-action endpoints are commands: they return the updated sales
		// order without `?include=` expansion. Clients that need expanded
		// sub-resources re-fetch via GET /sales-orders/{id}?include=...
	})
}
