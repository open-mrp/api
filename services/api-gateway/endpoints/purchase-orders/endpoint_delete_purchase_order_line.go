package purchaseorderep

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

// Request to delete a purchase order line.
type DeletePurchaseOrderLineRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// Purchase order line ID.
	PurchaseOrderLineID string `path:"line_id" validate:"required"`
}

// Deletes a purchase order line item and its related records.
//
// Any receiving order lines created for this line are deleted as well, so the quantity can no longer be received. Line numbers on the remaining lines are left as they are.
type DeletePurchaseOrderLineEndpoint struct{}

func (e *DeletePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePurchaseOrderLineRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeletePurchaseOrderLineRequest, *object.Deleted]{
		Title:             "Delete Purchase Order Line",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypePurchaseOrderLine,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate}}, Counterparty: authz.CounterpartyPermissions{Supplier: types.Permission{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePurchaseOrderLineRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PurchaseOrderSvc).DeletePurchaseOrderLine, constants.ObjectTypePurchaseOrderLine, "line_id")
		},
	})
}
