package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ReceiveReceivingOrderLineRequest is the request to receive a single receiving order line.
type ReceiveReceivingOrderLineRequest struct {
	// The ID of the receiving order.
	ReceivingOrderID string `path:"receivingOrderId" validate:"required"`
	// The ID of the receiving order line to receive.
	LineID string `path:"id" validate:"required"`
}

type ReceiveReceivingOrderLineEndpoint struct{}

func (e *ReceiveReceivingOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReceiveReceivingOrderLineRequest, *apiresource.ReceivingOrderLine] {
	return &apiendpoint.APIEndpoint[*ReceiveReceivingOrderLineRequest, *apiresource.ReceivingOrderLine]{
		Title:             "Receive Receiving Order Line",
		Description:       "Marks a single receiving order line as received.",
		Method:            http.MethodPut,
		Route:             "/v1/operations/receiving-orders/{receivingOrderId}/lines/{id}/actions/receive",
		Request:           &ReceiveReceivingOrderLineRequest{},
		Response:          &apiresource.ReceivingOrderLine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReceiveReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).ReceiveReceivingOrderLine
		},
	}
}
