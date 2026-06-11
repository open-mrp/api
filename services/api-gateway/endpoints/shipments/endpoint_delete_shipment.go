package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a shipment.
type DeleteShipmentRequest struct {
	// Shipment ID.
	ShipmentID string `path:"id" validate:"required"`
}

// Deletes a shipment along with its lines and shipping cases.
//
// Deleting a shipment also unpacks the associated pick lines and reopens the pick for the shipment's order so the items can be repacked.
type DeleteShipmentEndpoint struct{}

func (e *DeleteShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShipmentRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteShipmentRequest, *apiresource.EmptyResource]{
		Title:             "Delete Shipment",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipments/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShipmentRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShipmentSvc).DeleteShipment
		},
	})
}
