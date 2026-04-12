package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// VoidReceivingOrderRequest is the request to void a receiving order.
type VoidReceivingOrderRequest struct {
	// The ID of the receiving order to void.
	ReceivingOrderID string `path:"id" validate:"required"`
}

type VoidReceivingOrderEndpoint struct{}

func (e *VoidReceivingOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidReceivingOrderRequest, *apiresource.ReceivingOrder] {
	return &apiendpoint.APIEndpoint[*VoidReceivingOrderRequest, *apiresource.ReceivingOrder]{
		Title:             "Void Receiving Order",
		Description:       "Voids a receiving order, cancelling all of its lines.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{id}/actions/void",
		Request:           &VoidReceivingOrderRequest{},
		Response:          &apiresource.ReceivingOrder{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).VoidReceivingOrder
		},
	}
}
