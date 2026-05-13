package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a shipment line.
type RetrieveShipmentLineRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// Shipment line ID.
	ShipmentLineID string `path:"id" validate:"required"`
}

type RetrieveShipmentLineEndpoint struct{}

func (e *RetrieveShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShipmentLineRequest, *apiresource.ShipmentLine] {
	return &apiendpoint.APIEndpoint[*RetrieveShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:             "Retrieve Shipment Line",
		Description:       "Returns a shipment line by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments/{shipment_id}/lines/{id}",
		Request:           &RetrieveShipmentLineRequest{},
		Response:          &apiresource.ShipmentLine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).GetShipmentLine
		},
	}
}
