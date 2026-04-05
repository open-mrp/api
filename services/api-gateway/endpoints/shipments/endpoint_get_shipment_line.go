package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetShipmentLineRequest is the request to retrieve a single shipment line.
type GetShipmentLineRequest struct {
	// The ID of the shipment.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// The ID of the shipment line to retrieve.
	ShipmentLineID string `path:"id" validate:"required"`
}

type GetShipmentLineEndpoint struct{}

func (e *GetShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetShipmentLineRequest, *apiresource.ShipmentLine] {
	return &apiendpoint.APIEndpoint[*GetShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:             "Get Shipment Line",
		Description:       "Returns a single shipment line by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/shipments/{shipment_id}/lines/{id}",
		Request:           &GetShipmentLineRequest{},
		Response:          &apiresource.ShipmentLine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).GetShipmentLine
		},
	}
}
