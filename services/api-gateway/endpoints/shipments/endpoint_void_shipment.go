package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to void a shipment.
type VoidShipmentRequest struct {
	// ID of the shipment to void.
	ShipmentID string `path:"id" validate:"required"`
}

// Voids a shipped shipment, returning it to the `packed` status.
//
// Only shipments in the `shipped` status can be voided; otherwise a conflict error is returned. Voiding clears `shipped_at` and `shipped_by`, clears tracking and label details from the shipment's shipping cases, deletes the invoice created for the shipment if one exists, and marks the associated sales order as unfulfilled.
type VoidShipmentEndpoint struct{}

func (e *VoidShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidShipmentRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*VoidShipmentRequest, *apiresource.Shipment]{
		Title:             "Void Shipment",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/{id}/actions/void",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
			return svc.(ShipmentSvc).VoidShipment
		},
	})
}
