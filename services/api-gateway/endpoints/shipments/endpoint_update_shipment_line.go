package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateShipmentLineRequest is the request to partially update a shipment line.
type UpdateShipmentLineRequest struct {
	// The ID of the shipment.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// The ID of the shipment line to update.
	ShipmentLineID string `path:"id" validate:"required"`
	// The quantity value to set.
	QuantityValue *string `json:"quantity_value,omitempty" nullable:"false"`
	// The ID of the unit for the quantity.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
}

var sampleUpdateShipmentLineQuantityValue = "5.000000000000000000000000000000"
var sampleUpdateShipmentLineQuantityUnitID = apiresource.SampleUnitID
var sampleUpdateShipmentLineRequest = &UpdateShipmentLineRequest{
	QuantityValue:  &sampleUpdateShipmentLineQuantityValue,
	QuantityUnitID: &sampleUpdateShipmentLineQuantityUnitID,
}

func (*UpdateShipmentLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShipmentLineRequest)
}

type UpdateShipmentLineEndpoint struct{}

func (e *UpdateShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShipmentLineRequest, *apiresource.ShipmentLine] {
	return &apiendpoint.APIEndpoint[*UpdateShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:             "Update Shipment Line",
		Description:       "Partially updates a shipment line.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/shipments/{shipment_id}/lines/{id}",
		ContentType:       "application/json",
		Request:           &UpdateShipmentLineRequest{},
		Response:          &apiresource.ShipmentLine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).UpdateShipmentLine
		},
	}
}
