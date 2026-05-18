package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to receive all unstocked lines on a receiving order.
type ReceiveReceivingOrderRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"id" validate:"required"`
}

// Marks all unstocked lines on a receiving order as received.
type ReceiveReceivingOrderEndpoint struct{}

func (e *ReceiveReceivingOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReceiveReceivingOrderRequest, *apiresource.ReceivingOrder] {
	return (&apiendpoint.APIEndpoint[*ReceiveReceivingOrderRequest, *apiresource.ReceivingOrder]{
		Title:             "Receive Receiving Order",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{id}/actions/receive",
		Request:           &ReceiveReceivingOrderRequest{},
		Response:          &apiresource.ReceivingOrder{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReceiveReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).ReceiveReceivingOrder
		},
	}).WithDocSource(e)
}
