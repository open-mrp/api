package shipmentep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list lines for a shipment.
type ListShipmentLinesRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of lines for the specified shipment.
//
// A customer or supplier portal lists only the lines of a shipment on an order its own account placed; any other shipment is reported as not found.
type ListShipmentLinesEndpoint struct{}

func (e *ListShipmentLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListShipmentLinesRequest, *apiresource.List[apiresource.ShipmentLine]] {
	return (&apiendpoint.APIEndpoint[*ListShipmentLinesRequest, *apiresource.List[apiresource.ShipmentLine]]{
		Title:             "List Shipment Lines",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments/{shipment_id}/lines",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListShipmentLinesRequest) (*apiresource.List[apiresource.ShipmentLine], *apierror.APIError) {
			return svc.(ShipmentSvc).ListShipmentLines
		},
	})
}
