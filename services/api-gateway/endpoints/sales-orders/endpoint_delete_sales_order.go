package salesorderep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSalesOrderRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteSalesOrderRequest, *object.Deleted]{
		Title:             "Delete Sales Order",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeSalesOrder,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSalesOrderRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(SalesOrderSvc).DeleteSalesOrder, constants.ObjectTypeSalesOrder, "id")
		},
	})
}
