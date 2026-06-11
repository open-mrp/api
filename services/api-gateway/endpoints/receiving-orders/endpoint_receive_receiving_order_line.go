package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to receive a receiving order line.
type ReceiveReceivingOrderLineRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"receiving_order_id" validate:"required"`
	// Receiving order line ID.
	LineID string `path:"id" validate:"required"`
}

// Records the full outstanding quantity as received on a single receiving order line.
//
// Sets the line's quantity to the quantity still outstanding on its purchase order line (ordered minus previously received); if nothing is outstanding, the line is returned unchanged. This does not add inventory — use Stock Receiving Order to put the received quantity away.
type ReceiveReceivingOrderLineEndpoint struct{}

func (e *ReceiveReceivingOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReceiveReceivingOrderLineRequest, *apiresource.ReceivingOrderLine] {
	return (&apiendpoint.APIEndpoint[*ReceiveReceivingOrderLineRequest, *apiresource.ReceivingOrderLine]{
		Title:             "Receive Receiving Order Line",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{receiving_order_id}/lines/{id}/actions/receive",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeReceivingOrderLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReceiveReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).ReceiveReceivingOrderLine
		},
	})
}
