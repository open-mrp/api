package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a shipment line.
type UpdateShipmentLineRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// Shipment line ID.
	ShipmentLineID string `path:"id" validate:"required"`
	// Quantity value.
	QuantityValue *string `json:"quantity_value,omitempty" nullable:"false"`
	// Quantity unit ID.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" nullable:"false" validate:"omitempty"`
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

// Partially updates a shipment line.
type UpdateShipmentLineEndpoint struct{}

func (e *UpdateShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShipmentLineRequest, *apiresource.ShipmentLine] {
	return (&apiendpoint.APIEndpoint[*UpdateShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:             "Update Shipment Line",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/shipments/{shipment_id}/lines/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).UpdateShipmentLine
		},
	})
}
