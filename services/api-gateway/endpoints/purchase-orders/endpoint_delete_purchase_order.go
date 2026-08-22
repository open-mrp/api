package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a purchase order.
type DeletePurchaseOrderRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
}

// Deletes a purchase order along with its lines, email contacts, and receiving order.
//
// Orders in `fulfilled` status cannot be deleted; re-open the order first. Deleting is permanent, and a later request for the same order reports that it has already been deleted rather than that it was never found.
type DeletePurchaseOrderEndpoint struct{}

func (e *DeletePurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePurchaseOrderRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeletePurchaseOrderRequest, *apiresource.EmptyResource]{
		Title:             "Delete Purchase Order",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePurchaseOrderRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).DeletePurchaseOrder
		},
	})
}
