package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to mark a shipment as shipped.
type ShipShipmentRequest struct {
	// ID of the shipment to ship.
	ShipmentID string `path:"id" validate:"required"`
	// Whether to email the customer a shipping notification.
	//
	// Whether to email the customer the invoice raised for this shipment.
	EmailCustomer bool `json:"email_customer"`
}

var sampleShipShipmentRequest = &ShipShipmentRequest{
	EmailCustomer: true,
}

func (*ShipShipmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleShipShipmentRequest)
}

// Dispatches a packed shipment, marking it and its cases as shipped.
//
// Sets the shipment status to `shipped`, records `shipped_at` and the acting user as `shipped_by`, marks all shipping cases as shipped, and assigns an SSCC to any case that does not already have one. Fails with a conflict error if the shipment has already been shipped, so shipping is a one-way move that can only be reversed with the void action.
type ShipShipmentEndpoint struct{}

func (e *ShipShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*ShipShipmentRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*ShipShipmentRequest, *apiresource.Shipment]{
		Title:               "Ship Shipment",
		Method:              http.MethodPost,
		Route:               "/v1/operations/shipments/{id}/actions/ship",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ShipShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
			return svc.(ShipmentSvc).ShipShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "related.sales_order", "customer", "freight", "shipping_address", "shipped_by", "related.invoice", "related.pick"},
		}),
	})
}
