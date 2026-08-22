package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a shipment line.
type DeleteShipmentLineRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// Shipment line ID.
	ShipmentLineID string `path:"id" validate:"required"`
}

// Removes a line from a shipment.
//
// Unlike deleting the whole shipment, removing a single line leaves the pick for the order untouched, so the pick's lines keep their existing packed state.
type DeleteShipmentLineEndpoint struct{}

func (e *DeleteShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShipmentLineRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteShipmentLineRequest, *apiresource.EmptyResource]{
		Title:               "Delete Shipment Line",
		Method:              http.MethodDelete,
		Route:               "/v1/operations/shipments/{shipment_id}/lines/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShipmentLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShipmentSvc).DeleteShipmentLine
		},
	})
}
