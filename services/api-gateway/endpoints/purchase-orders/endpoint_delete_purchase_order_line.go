package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeletePurchaseOrderLineRequest is the request to delete a purchase order line.
type DeletePurchaseOrderLineRequest struct {
	// The ID of the purchase order.
	PurchaseOrderID string `path:"id" validate:"required"`
	// The ID of the purchase order line to delete.
	PurchaseOrderLineID string `path:"lineId" validate:"required"`
}

type DeletePurchaseOrderLineEndpoint struct{}

func (e *DeletePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePurchaseOrderLineRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeletePurchaseOrderLineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Purchase Order Line",
		Description:       "Deletes a purchase order line item and its related records.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/purchase-orders/{id}/lines/{lineId}",
		Request:           &DeletePurchaseOrderLineRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePurchaseOrderLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).DeletePurchaseOrderLine
		},
	}
}
