package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a sales order line.
type DeleteSalesOrderLineRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Sales order line ID.
	SalesOrderLineID string `path:"lineId" validate:"required"`
}

type DeleteSalesOrderLineEndpoint struct{}

func (e *DeleteSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSalesOrderLineRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteSalesOrderLineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Sales Order Line",
		Description:       "Deletes a sales order line and related records.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines/{lineId}",
		Request:           &DeleteSalesOrderLineRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSalesOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SalesOrderSvc).DeleteSalesOrderLine
		},
	}
}
