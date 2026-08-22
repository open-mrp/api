package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a sales order.
type DeleteSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// Deletes a sales order and all its related records.
//
// Removes the order's lines, pick, shipment and invoice lines, and email contacts, and releases any inventory it had reserved. Fulfilled orders cannot be deleted.
type DeleteSalesOrderEndpoint struct{}

func (e *DeleteSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSalesOrderRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteSalesOrderRequest, *apiresource.EmptyResource]{
		Title:             "Delete Sales Order",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSalesOrderRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SalesOrderSvc).DeleteSalesOrder
		},
	})
}
