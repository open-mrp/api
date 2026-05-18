package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to void a receiving order.
type VoidReceivingOrderRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"id" validate:"required"`
}

// Voids a receiving order, cancelling all lines.
type VoidReceivingOrderEndpoint struct{}

func (e *VoidReceivingOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidReceivingOrderRequest, *apiresource.ReceivingOrder] {
	return (&apiendpoint.APIEndpoint[*VoidReceivingOrderRequest, *apiresource.ReceivingOrder]{
		Title:             "Void Receiving Order",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{id}/actions/void",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).VoidReceivingOrder
		},
	})
}
