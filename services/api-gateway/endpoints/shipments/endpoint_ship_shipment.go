package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to mark a shipment as shipped.
type ShipShipmentRequest struct {
	// Shipment ID.
	ShipmentID string `path:"id" validate:"required"`
	// Whether to email the customer a shipping notification.
	EmailCustomer bool `json:"email_customer"`
}

var sampleShipShipmentRequest = &ShipShipmentRequest{
	EmailCustomer: true,
}

func (*ShipShipmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleShipShipmentRequest)
}

// Marks a shipment as shipped and optionally sends a shipping notification email to the customer.
type ShipShipmentEndpoint struct{}

func (e *ShipShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*ShipShipmentRequest, *apiresource.ShipmentDetail] {
	return (&apiendpoint.APIEndpoint[*ShipShipmentRequest, *apiresource.ShipmentDetail]{
		Title:             "Ship Shipment",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipments/{id}/actions/ship",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ShipShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
			return svc.(ShipmentSvc).ShipShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "sales_order", "customer", "carrier", "service_level", "shipping_address", "shipped_by", "invoice", "pick"},
		}),
	})
}
