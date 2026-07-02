package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a receiving order line's quantity.
type UpdateReceivingOrderLineRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"receiving_order_id" validate:"required"`
	// Receiving order line ID.
	LineID string `path:"id" validate:"required"`
	// New received quantity for the line, as a decimal string.
	//
	// When omitted, the line is returned unchanged.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero"`
}

var sampleUpdateReceivingOrderLineQuantityValue = "50"
var sampleUpdateReceivingOrderLineRequest = &UpdateReceivingOrderLineRequest{
	QuantityValue: field.Some(sampleUpdateReceivingOrderLineQuantityValue),
}

func (*UpdateReceivingOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateReceivingOrderLineRequest)
}

// Updates the received quantity on a receiving order line.
//
// Use this to record the quantity actually received — for example a partial delivery — before stocking the order.
type UpdateReceivingOrderLineEndpoint struct{}

func (e *UpdateReceivingOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateReceivingOrderLineRequest, *apiresource.ReceivingOrderLine] {
	return (&apiendpoint.APIEndpoint[*UpdateReceivingOrderLineRequest, *apiresource.ReceivingOrderLine]{
		Title:             "Update Receiving Order Line",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/receiving-orders/{receiving_order_id}/lines/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeReceivingOrderLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).UpdateReceivingOrderLine
		},
	})
}
