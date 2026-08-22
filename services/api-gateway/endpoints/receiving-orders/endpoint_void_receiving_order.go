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

// Request to void a receiving order.
type VoidReceivingOrderRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"id" validate:"required"`
}

// Voids a receiving order, resetting all receiving progress.
//
// Every line's received quantity is reset to `0` and its stocked state is cleared, the extra lines created for short receipts are removed so that one line per purchase order line remains, and the order returns to open. The receiving order itself is not deleted, and it can be received and stocked again from scratch.
//
// A receiving order that has already been marked complete is only reopened: the extra lines are still removed, but the lines that remain keep their received quantities and stay marked as stocked.
//
// Deliveries and inventory received by earlier stocking are not reversed — voiding only reopens the receiving order.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeReceivingOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidReceivingOrderRequest) (*apiresource.ReceivingOrder, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).VoidReceivingOrder
		},
	})
}
