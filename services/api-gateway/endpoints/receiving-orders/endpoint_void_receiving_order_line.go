package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// VoidReceivingOrderLineRequest is the request to void a single receiving order line.
type VoidReceivingOrderLineRequest struct {
	// The ID of the receiving order.
	ReceivingOrderID string `path:"receivingOrderId" validate:"required"`
	// The ID of the receiving order line to void.
	LineID string `path:"id" validate:"required"`
}

type VoidReceivingOrderLineEndpoint struct{}

func (e *VoidReceivingOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidReceivingOrderLineRequest, *apiresource.ReceivingOrderLine] {
	return &apiendpoint.APIEndpoint[*VoidReceivingOrderLineRequest, *apiresource.ReceivingOrderLine]{
		Title:             "Void Receiving Order Line",
		Description:       "Voids a single receiving order line.",
		Method:            http.MethodPut,
		Route:             "/v1/operations/receiving-orders/{receivingOrderId}/lines/{id}/actions/void",
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
