package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a purchase order.
type DeletePurchaseOrderRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
}

// Deletes a purchase order and all its related records.
//
// Orders in `fulfilled` status cannot be deleted.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePurchaseOrderRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).DeletePurchaseOrder
		},
	})
}
