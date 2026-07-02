package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to receive all unstocked lines on a receiving order.
type ReceiveReceivingOrderRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"id" validate:"required"`
}

// Records the full outstanding quantity as received on every unstocked line of a receiving order.
//
// Each unstocked line's quantity is set to the quantity still outstanding on its purchase order line (ordered minus previously received); lines with nothing outstanding are left unchanged. This does not add inventory — use Stock Receiving Order to put the received quantities away.
type ReceiveReceivingOrderEndpoint struct{}

func (e *ReceiveReceivingOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReceiveReceivingOrderRequest, *apiresource.ReceivingOrder] {
	return (&apiendpoint.APIEndpoint[*ReceiveReceivingOrderRequest, *apiresource.ReceivingOrder]{
		Title:             "Receive Receiving Order",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{id}/actions/receive",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeReceivingOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReceiveReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).ReceiveReceivingOrder
		},
	})
}
