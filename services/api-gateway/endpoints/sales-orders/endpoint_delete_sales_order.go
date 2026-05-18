package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a sales order.
type DeleteSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// Deletes a sales order and all its related records.
type DeleteSalesOrderEndpoint struct{}

func (e *DeleteSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSalesOrderRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteSalesOrderRequest, *apiresource.EmptyResource]{
		Title:             "Delete Sales Order",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSalesOrderRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SalesOrderSvc).DeleteSalesOrder
		},
	})
}
