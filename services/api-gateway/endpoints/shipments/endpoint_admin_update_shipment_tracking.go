package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// ! separate endpoint due to SDK type resttrictions
// Request to correct a shipped shipment's tracking and routing.
type AdminUpdateShipmentTrackingRequest struct {
	// ID of the shipment to correct.
	ShipmentID string `path:"id" validate:"required"`
	// Carrier master tracking number covering the shipment as a whole.
	MasterTrackingNumber field.Optional[string] `json:"master_tracking_number,omitzero" validate:"omitempty,max=255"`
	// ID of the carrier that actually carried the shipment; the shipment's cases move with it.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// ID of the carrier service level the shipment actually travelled on.
	// Sending this without `carrier_id` keeps the existing carrier, so the service level should belong to that carrier; send `null` to drop the service level entirely.
	ServiceLevelID field.Clearable[string] `json:"service_level_id,omitzero" validate:"omitempty"`
}

var sampleAdminUpdateShipmentTrackingRequest = &AdminUpdateShipmentTrackingRequest{
	MasterTrackingNumber: field.Some("1Z999AA10123456784"),
}

func (*AdminUpdateShipmentTrackingRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAdminUpdateShipmentTrackingRequest)
}

// Rewrites the carrier, service level and master tracking number of a shipment that has already shipped.
// Administrators only: the ordinary update refuses to re-route a dispatched shipment, and this is the deliberate override for one that went out mis-routed.
type AdminUpdateShipmentTrackingEndpoint struct{}

func (e *AdminUpdateShipmentTrackingEndpoint) Materialize() *apiendpoint.APIEndpoint[*AdminUpdateShipmentTrackingRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*AdminUpdateShipmentTrackingRequest, *apiresource.Shipment]{
		Title:               "Admin Update Shipment Tracking",
		Method:              http.MethodPost,
		Route:               "/v1/operations/shipments/{id}/actions/admin-update-tracking",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AdminUpdateShipmentTrackingRequest) (*apiresource.Shipment, *apierror.APIError) {
			return svc.(ShipmentSvc).AdminUpdateShipmentTracking
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "related.sales_order", "customer", "freight", "shipping_address", "shipped_by", "related.invoice", "related.pick"},
		}),
	})
}
