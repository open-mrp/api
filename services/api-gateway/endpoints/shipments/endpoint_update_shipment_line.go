package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a shipment line.
type UpdateShipmentLineRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// Shipment line ID.
	ShipmentLineID string `path:"id" validate:"required"`
	// Quantity shipped, as a decimal string.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero"`
	// ID of the unit of measure for `quantity_value`.
	QuantityUnitID field.Optional[string] `json:"quantity_unit_id,omitzero" validate:"omitempty"`
}

var sampleUpdateShipmentLineRequest = &UpdateShipmentLineRequest{
	QuantityValue:  field.Some("5.000000000000000000000000000000"),
	QuantityUnitID: field.Some(apiresource.SampleUnitID),
}

func (*UpdateShipmentLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShipmentLineRequest)
}

// Partially updates a shipment line.
type UpdateShipmentLineEndpoint struct{}

func (e *UpdateShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShipmentLineRequest, *apiresource.ShipmentLine] {
	return (&apiendpoint.APIEndpoint[*UpdateShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:               "Update Shipment Line",
		Method:              http.MethodPatch,
		Route:               "/v1/operations/shipments/{shipment_id}/lines/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).UpdateShipmentLine
		},
	})
}
