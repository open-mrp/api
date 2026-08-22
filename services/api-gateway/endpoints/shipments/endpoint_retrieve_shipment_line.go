package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a shipment line.
type RetrieveShipmentLineRequest struct {
	// Shipment ID.
	ShipmentID string `path:"shipment_id" validate:"required"`
	// Shipment line ID.
	ShipmentLineID string `path:"id" validate:"required"`
}

// Returns a shipment line by ID.
type RetrieveShipmentLineEndpoint struct{}

func (e *RetrieveShipmentLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShipmentLineRequest, *apiresource.ShipmentLine] {
	return (&apiendpoint.APIEndpoint[*RetrieveShipmentLineRequest, *apiresource.ShipmentLine]{
		Title:               "Retrieve Shipment Line",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/shipments/{shipment_id}/lines/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
			return svc.(ShipmentSvc).GetShipmentLine
		},
	})
}
