package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to void a receiving order line.
type VoidReceivingOrderLineRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"receiving_order_id" validate:"required"`
	// Receiving order line ID.
	LineID string `path:"id" validate:"required"`
}

type VoidReceivingOrderLineEndpoint struct{}

func (e *VoidReceivingOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidReceivingOrderLineRequest, *apiresource.ReceivingOrderLine] {
	return &apiendpoint.APIEndpoint[*VoidReceivingOrderLineRequest, *apiresource.ReceivingOrderLine]{
		Title:             "Void Receiving Order Line",
		Description:       "Voids a receiving order line.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{receiving_order_id}/lines/{id}/actions/void",
		Request:           &VoidReceivingOrderLineRequest{},
		Response:          &apiresource.ReceivingOrderLine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).VoidReceivingOrderLine
		},
	}
}
