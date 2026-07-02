package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a line on a purchase order.
type CreatePurchaseOrderLineRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	apirequest.OrderLineInput
}

// Creates a line item on a purchase order.
//
// If the order has already been issued, a matching receiving order line is created as well.
type CreatePurchaseOrderLineEndpoint struct{}

func (e *CreatePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLine] {
	return (&apiendpoint.APIEndpoint[*CreatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLine]{
		Title:             "Create Purchase Order Line",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}/lines",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLine, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).CreatePurchaseOrderLine
		},
	})
}
