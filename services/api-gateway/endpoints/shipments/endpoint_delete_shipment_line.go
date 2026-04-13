package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a shipment line.
type DeleteShipmentLineRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// Shipment line ID.
	ShipmentLineID string `path:"id" validate:"required"`
}

type DeleteShipmentLineEndpoint struct{}

func (e *DeleteShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShipmentLineRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteShipmentLineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Shipment Line",
		Description:       "Deletes a line from a shipment.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipments/{shipment_id}/lines/{id}",
		ContentType:       "application/json",
		Request:           &DeleteShipmentLineRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShipmentLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShipmentSvc).DeleteShipmentLine
		},
	}
}
