package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to void a shipment.
type VoidShipmentRequest struct {
	// ID of the shipment to void.
	ShipmentID string `path:"id" validate:"required"`
}

// Voids a shipped shipment, returning it to the `packed` status so it can be corrected and shipped again.
//
// Only shipments in the `shipped` status can be voided; otherwise a conflict error is returned. Voiding clears `shipped_at`, `shipped_by` and the master tracking number, clears the shipped timestamp, tracking number and label on every shipping case and resets each case's freight charge to zero, deletes the invoice raised for the shipment if one exists, and returns the associated sales order to its unfulfilled state. Case SSCCs are kept.
type VoidShipmentEndpoint struct{}

func (e *VoidShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidShipmentRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*VoidShipmentRequest, *apiresource.Shipment]{
		Title:               "Void Shipment",
		Method:              http.MethodPost,
		Route:               "/v1/operations/shipments/{id}/actions/void",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
			return svc.(ShipmentSvc).VoidShipment
		},
	})
}
