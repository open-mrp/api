package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteSalesOrderRequest is the request to delete a sales order.
type DeleteSalesOrderRequest struct {
	// The ID of the sales order to delete.
	SalesOrderID string `path:"id" validate:"required"`
}

type DeleteSalesOrderEndpoint struct{}

func (e *DeleteSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSalesOrderRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteSalesOrderRequest, *apiresource.EmptyResource]{
		Title:             "Delete Sales Order",
		Description:       "Deletes a sales order and all its related records.",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/sales-orders/{id}",
		Request:           &DeleteSalesOrderRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSalesOrderRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SalesOrderSvc).DeleteSalesOrder
		},
	}
}
