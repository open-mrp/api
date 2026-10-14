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

// Request to delete a shipment.
type DeleteShipmentRequest struct {
	// Shipment ID.
	ShipmentID string `path:"id" validate:"required"`
}

// Deletes a shipment along with its lines and shipping cases.
//
// Deleting a shipment also unpacks the associated pick lines and reopens the pick for the shipment's order so the items can be repacked. A shipped shipment cannot be deleted; void it first, or a conflict error is returned.
type DeleteShipmentEndpoint struct{}

func (e *DeleteShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShipmentRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteShipmentRequest, *object.Deleted]{
		Title:             "Delete Shipment",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipments/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeShipment,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShipmentRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ShipmentSvc).DeleteShipment, constants.ObjectTypeShipment, "id")
		},
	})
}
