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

func (e *DeleteSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSalesOrderLineRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteSalesOrderLineRequest, *object.Deleted]{
		Title:             "Delete Sales Order Line",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSalesOrderLineRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(SalesOrderSvc).DeleteSalesOrderLine, constants.ObjectTypeSalesOrderLine, "line_id")
		},
	})
}
