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

// Request to delete a purchase order.
type DeletePurchaseOrderRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
}

// Deletes a purchase order along with its lines, email contacts, and receiving order.
//
// Orders in `fulfilled` status cannot be deleted; re-open the order first. Deleting is permanent, and a later request for the same order reports that it has already been deleted rather than that it was never found.
type DeletePurchaseOrderEndpoint struct{}

func (e *DeletePurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePurchaseOrderRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeletePurchaseOrderRequest, *object.Deleted]{
		Title:             "Delete Purchase Order",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypePurchaseOrder,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePurchaseOrderRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PurchaseOrderSvc).DeletePurchaseOrder, constants.ObjectTypePurchaseOrder, "id")
		},
	})
}
