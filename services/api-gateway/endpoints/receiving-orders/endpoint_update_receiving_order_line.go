package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateReceivingOrderLineRequest is the request to update a receiving order line's quantity.
type UpdateReceivingOrderLineRequest struct {
	// The ID of the receiving order.
	ReceivingOrderID string `path:"receivingOrderId" validate:"required"`
	// The ID of the receiving order line to update.
	LineID string `path:"id" validate:"required"`
	// The quantity value to set for this line.
	QuantityValue *string `json:"quantity_value,omitempty"`
}

var sampleUpdateReceivingOrderLineQuantityValue = "50"
var sampleUpdateReceivingOrderLineRequest = &UpdateReceivingOrderLineRequest{
	QuantityValue: &sampleUpdateReceivingOrderLineQuantityValue,
}

func (*UpdateReceivingOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateReceivingOrderLineRequest)
}

type UpdateReceivingOrderLineEndpoint struct{}

func (e *UpdateReceivingOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateReceivingOrderLineRequest, *apiresource.ReceivingOrderLine] {
	return &apiendpoint.APIEndpoint[*UpdateReceivingOrderLineRequest, *apiresource.ReceivingOrderLine]{
		Title:             "Update Receiving Order Line",
		Description:       "Partially updates a receiving order line's quantity value.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/receiving-orders/{receivingOrderId}/lines/{id}",
		ContentType:       "application/json",
		Request:           &UpdateReceivingOrderLineRequest{},
		Response:          &apiresource.ReceivingOrderLine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateReceivingOrderLineRequest) (*apiresource.ReceivingOrderLine, *apierror.APIError) {
			return svc.(ReceivingOrderSvc).UpdateReceivingOrderLine
		},
	}
}
