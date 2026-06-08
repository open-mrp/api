package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a shipment.
type UpdateShipmentRequest struct {
	// Shipment ID.
	ShipmentID string `path:"id" validate:"required"`
	// Note for the shipment.
	Note field.Optional[string] `json:"note,omitzero"`
	// Shipment number.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Master tracking number.
	MasterTrackingNumber field.Optional[string] `json:"master_tracking_number,omitzero" validate:"omitempty,max=255"`
	// Carrier ID.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// Service level ID.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero" validate:"omitempty"`
}

var sampleUpdateShipmentRequest = &UpdateShipmentRequest{
	Note: field.Some("Updated shipping note"),
}

func (*UpdateShipmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShipmentRequest)
}

// Partially updates a shipment.
type UpdateShipmentEndpoint struct{}

func (e *UpdateShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShipmentRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*UpdateShipmentRequest, *apiresource.Shipment]{
		Title:             "Update Shipment",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/shipments/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
			return svc.(ShipmentSvc).UpdateShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "sales_order", "customer", "freight", "shipping_address", "shipped_by", "invoice", "pick"},
		}),
	})
}
