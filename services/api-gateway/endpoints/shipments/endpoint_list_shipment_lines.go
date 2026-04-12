package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListShipmentLinesRequest is the request to list lines for a shipment.
type ListShipmentLinesRequest struct {
	// The ID of the shipment to list lines for.
	ShipmentID string `path:"shipment_id" validate:"required"`
	apiresource.PaginationRequest
}

type ListShipmentLinesEndpoint struct{}

func (e *ListShipmentLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListShipmentLinesRequest, *apiresource.List[apiresource.ShipmentLine]] {
	return &apiendpoint.APIEndpoint[*ListShipmentLinesRequest, *apiresource.List[apiresource.ShipmentLine]]{
		Title:             "List Shipment Lines",
		Description:       "Returns a paginated list of lines for the specified shipment.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments/{shipment_id}/lines",
		Request:           &ListShipmentLinesRequest{},
		Response:          &apiresource.List[apiresource.ShipmentLine]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListShipmentLinesRequest) (*apiresource.List[apiresource.ShipmentLine], *apierror.APIError) {
			return svc.(ShipmentSvc).ListShipmentLines
		},
	}
}
