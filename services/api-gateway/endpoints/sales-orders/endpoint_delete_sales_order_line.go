package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a sales order line.
type DeleteSalesOrderLineRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Sales order line ID.
	SalesOrderLineID string `path:"line_id" validate:"required"`
}

// Deletes a sales order line and its pick lines.
//
// A line cannot be removed once it has been packed onto a shipment, or once the order is fulfilled, and removing one from an order that is already completed or has a shipped shipment requires an admin. The remaining lines are renumbered so the sequence stays contiguous, and if this was the last line left to pick, the order's pick is deleted and the order falls back to `estimate` with its reserved inventory released.
type DeleteSalesOrderLineEndpoint struct{}

func (e *DeleteSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSalesOrderLineRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteSalesOrderLineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Sales Order Line",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSalesOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SalesOrderSvc).DeleteSalesOrderLine
		},
	})
}
