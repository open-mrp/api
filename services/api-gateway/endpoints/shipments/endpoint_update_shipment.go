package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update a shipment.
type UpdateShipmentRequest struct {
	// ID of the shipment to update.
	ShipmentID string `path:"id" validate:"required"`
	// Note for the shipment.
	Note field.Optional[string] `json:"note,omitzero"`
	// Human-readable shipment number.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Carrier master tracking number covering the shipment as a whole.
	MasterTrackingNumber field.Optional[string] `json:"master_tracking_number,omitzero" validate:"omitempty,max=255"`
	// ID of the carrier to set on the shipment's freight.
	//
	// Changing the carrier records the new selection only; it does not re-rate the shipment, so the freight charges already recorded on the shipping cases are left as they are.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// ID of the carrier service level to set on the shipment's freight.
	//
	// Sending this without `carrier_id` keeps the existing carrier, so the service level should belong to that carrier; send `null` to drop the service level entirely.
	ServiceLevelID field.Clearable[string] `json:"service_level_id,omitzero" validate:"omitempty"`
}

var sampleUpdateShipmentRequest = &UpdateShipmentRequest{
	Note: field.Some("Updated shipping note"),
}

func (*UpdateShipmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShipmentRequest)
}

// Updates a shipment's paperwork details and carrier selection.
//
// Only the fields sent are changed. A shipment's status is not editable here: use the ship and void actions to move a shipment between `packed` and `shipped`.
type UpdateShipmentEndpoint struct{}

func (e *UpdateShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShipmentRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*UpdateShipmentRequest, *apiresource.Shipment]{
		Title:               "Update Shipment",
		Method:              http.MethodPatch,
		Route:               "/v1/operations/shipments/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
			return svc.(ShipmentSvc).UpdateShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "related.sales_order", "customer", "freight", "shipping_address", "shipped_by", "related.invoice", "related.pick"},
		}),
	})
}
