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
	// Shipment ID.
	ShipmentID string `path:"id" validate:"required"`
}

// Voids a shipment, cancelling it and returning its lines to the sales order.
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
