package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list lines for a shipment.
type ListShipmentLinesRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	apiresource.PaginationRequest
}

// Returns a paginated list of lines for the specified shipment.
type ListShipmentLinesEndpoint struct{}

func (e *ListShipmentLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListShipmentLinesRequest, *apiresource.List[apiresource.ShipmentLine]] {
	return (&apiendpoint.APIEndpoint[*ListShipmentLinesRequest, *apiresource.List[apiresource.ShipmentLine]]{
		Title:               "List Shipment Lines",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/shipments/{shipment_id}/lines",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListShipmentLinesRequest) (*apiresource.List[apiresource.ShipmentLine], *apierror.APIError) {
			return svc.(ShipmentSvc).ListShipmentLines
		},
	})
}
