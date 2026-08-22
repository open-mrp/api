package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
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

func (e *DeletePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePurchaseOrderLineRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeletePurchaseOrderLineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Purchase Order Line",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePurchaseOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).DeletePurchaseOrderLine
		},
	})
}
