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

// Request to void a receiving order line.
type VoidReceivingOrderLineRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"receiving_order_id" validate:"required"`
	// Receiving order line ID.
	LineID string `path:"id" validate:"required"`
}

// Voids a single receiving order line, resetting its receiving progress.
//
// The line's received quantity is reset to `0` and its stocked state is cleared, leaving the rest of the order untouched. The line itself is not deleted, and any inventory already stocked from it is not reversed.
type VoidReceivingOrderLineEndpoint struct{}

func (e *VoidReceivingOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidReceivingOrderLineRequest, *apiresource.ReceivingOrderLine] {
	return (&apiendpoint.APIEndpoint[*VoidReceivingOrderLineRequest, *apiresource.ReceivingOrderLine]{
		Title:             "Void Receiving Order Line",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders/{receiving_order_id}/lines/{id}/actions/void",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeReceivingOrderLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).VoidReceivingOrderLine
		},
	})
}
