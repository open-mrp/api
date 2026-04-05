package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteShipmentRequest is the request to delete a shipment.
type DeleteShipmentRequest struct {
	// The ID of the shipment to delete.
	ShipmentID string `path:"id" validate:"required"`
}

type DeleteShipmentEndpoint struct{}

func (e *DeleteShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShipmentRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteShipmentRequest, *apiresource.EmptyResource]{
		Title:             "Delete Shipment",
		Description:       "Deletes a shipment. Cannot be deleted if it has already been shipped.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipments/{id}",
		ContentType:       "application/json",
		Request:           &DeleteShipmentRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShipmentRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShipmentSvc).DeleteShipment
		},
	}
}
