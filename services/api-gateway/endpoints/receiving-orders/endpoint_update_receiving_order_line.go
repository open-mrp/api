package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a receiving order line's quantity.
type UpdateReceivingOrderLineRequest struct {
	// Receiving order ID.
	ReceivingOrderID string `path:"receiving_order_id" validate:"required"`
	// Receiving order line ID.
	LineID string `path:"id" validate:"required"`
	// New received quantity for the line, in any unit of the line's item. Must not be negative.
	//
	// The line takes the given unit as well as the value. When omitted, the line is returned unchanged.
	Quantity field.Optional[apirequest.QuantityInput] `json:"quantity,omitzero"`
}

var sampleUpdateReceivingOrderLineRequest = &UpdateReceivingOrderLineRequest{
	Quantity: field.Some(apirequest.QuantityInput{Value: "50", UnitID: apiresource.SampleUnitID}),
}

func (*UpdateReceivingOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateReceivingOrderLineRequest)
}

// Updates the received quantity on a receiving order line.
//
// Use this to record the quantity that actually arrived — a partial delivery, for example — before stocking the order. Nothing enters inventory until the order is stocked.
//
// A line that has been stocked, or a line of a completed order, cannot be changed.
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
		IncludeConfig: receivingOrderLineIncludes(),
	})
}
