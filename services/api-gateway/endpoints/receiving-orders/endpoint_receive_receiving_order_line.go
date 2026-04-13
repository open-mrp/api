package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to receive a receiving order line.
type ReceiveReceivingOrderLineRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"receivingOrderId" validate:"required"`
	// Receiving order line ID.
	LineID string `path:"id" validate:"required"`
}

type ReceiveReceivingOrderLineEndpoint struct{}

func (e *ReceiveReceivingOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReceiveReceivingOrderLineRequest, *apiresource.ReceivingOrderLine] {
	return &apiendpoint.APIEndpoint[*ReceiveReceivingOrderLineRequest, *apiresource.ReceivingOrderLine]{
		Title:             "Receive Receiving Order Line",
		Description:       "Marks a receiving order line as received.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
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
