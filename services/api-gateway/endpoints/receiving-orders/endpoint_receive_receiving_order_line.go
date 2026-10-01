package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
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
// Sets the line's quantity, in the ordered unit, to the ordered quantity less what the purchase order line's other receiving lines already hold, so that together they cover the order. A line that already holds at least that much, or whose order line is already covered, is returned unchanged. Nothing enters inventory; use Stock Receiving Order to put the received quantity away.
//
// A line that has been stocked, or a line of a completed order, cannot be received.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeReceivingOrderLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReceiveReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).ReceiveReceivingOrderLine
		},
		IncludeConfig: receivingOrderLineIncludes(),
	})
}
