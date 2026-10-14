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
)

// Request to retrieve a shipment by ID.
type RetrieveShipmentRequest struct {
	// Shipment ID.
	ShipmentID string `path:"id" validate:"required"`
}

// Returns a shipment by ID.
//
// A customer or supplier portal retrieves only a shipment on an order its own account placed; any other shipment is reported as not found.
type RetrieveShipmentEndpoint struct{}

func (e *RetrieveShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShipmentRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*RetrieveShipmentRequest, *apiresource.Shipment]{
		Title:             "Retrieve Shipment",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		Preview:           true,
		ObjectType:        constants.ObjectTypeShipment,
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
				"lines.item",
				"lines.quantity",
				"lines.quantity.unit",
				"lines.sales_order_line",
				"lines.sales_order_line.product",
			},
		}),
	})
}
