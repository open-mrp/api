package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreatePurchaseOrderLineRequest is the request to create a new line on a purchase order.
type CreatePurchaseOrderLineRequest struct {
	// The ID of the purchase order.
	PurchaseOrderID string `path:"id" validate:"required"`
	apirequest.OrderLineInput
}

type CreatePurchaseOrderLineEndpoint struct{}

func (e *CreatePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLineDetail] {
	return &apiendpoint.APIEndpoint[*CreatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLineDetail]{
		Title:             "Create Purchase Order Line",
		Description:       "Creates a new line item on a purchase order.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/purchase-orders/{id}/lines",
		Request:           &CreatePurchaseOrderLineRequest{},
		Response:          &apiresource.PurchaseOrderLineDetail{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLineDetail, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).CreatePurchaseOrderLine
		},
	}
}
