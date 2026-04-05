package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// VoidShipmentRequest is the request to void a shipment.
type VoidShipmentRequest struct {
	// The ID of the shipment to void.
	ShipmentID string `path:"id" validate:"required"`
}

type VoidShipmentEndpoint struct{}

func (e *VoidShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*VoidShipmentRequest, *apiresource.ShipmentDetail] {
	return &apiendpoint.APIEndpoint[*VoidShipmentRequest, *apiresource.ShipmentDetail]{
		Title:             "Void Shipment",
		Description:       "Voids a shipment, cancelling it and returning its lines to the sales order.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/{id}/actions/void",
		ContentType:       "application/json",
		Request:           &VoidShipmentRequest{},
		Response:          &apiresource.ShipmentDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VoidShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
			return svc.(ShipmentSvc).VoidShipment
		},
	}
}
