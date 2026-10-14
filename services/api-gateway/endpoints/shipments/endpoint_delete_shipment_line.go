package shipmentep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShipmentLineRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteShipmentLineRequest, *object.Deleted]{
		Title:             "Delete Shipment Line",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipments/{shipment_id}/lines/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeShipmentLine,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShipmentLineRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ShipmentSvc).DeleteShipmentLine, constants.ObjectTypeShipmentLine, "id")
		},
	})
}
