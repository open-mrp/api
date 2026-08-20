package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a shipment by ID.
type RetrieveShipmentRequest struct {
	// Shipment ID.
	ShipmentID string `path:"id" validate:"required"`
}

// Returns a shipment by ID.
type RetrieveShipmentEndpoint struct{}

func (e *RetrieveShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShipmentRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*RetrieveShipmentRequest, *apiresource.Shipment]{
		Title:               "Retrieve Shipment",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/shipments/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
			return svc.(ShipmentSvc).GetShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields: []string{
				"related.sales_order",
				"customer",
				"freight",
				"shipping_address",
				"shipped_by",
				"related.invoice",
				"related.pick",
				"shipping_cases",
				"lines",
				"lines.sales_order_line",
				"lines.sales_order_line.product",
			},
		}),
	})
}
