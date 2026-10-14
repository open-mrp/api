package shipmentep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a shipment line.
type RetrieveShipmentLineRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// Shipment line ID.
	ShipmentLineID string `path:"id" validate:"required"`
}

// Returns a shipment line by ID.
//
// A customer or supplier portal retrieves only a line of a shipment on an order its own account placed; any other shipment is reported as not found.
type RetrieveShipmentLineEndpoint struct{}

func (e *RetrieveShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShipmentLineRequest, *apiresource.ShipmentLine] {
	return (&apiendpoint.APIEndpoint[*RetrieveShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:             "Retrieve Shipment Line",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments/{shipment_id}/lines/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).GetShipmentLine
		},
	})
}
