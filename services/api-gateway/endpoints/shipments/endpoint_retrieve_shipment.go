package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
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

func (e *RetrieveShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShipmentRequest, *apiresource.ShipmentDetail] {
	return (&apiendpoint.APIEndpoint[*RetrieveShipmentRequest, *apiresource.ShipmentDetail]{
		Title:             "Retrieve Shipment",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
			return svc.(ShipmentSvc).GetShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "sales_order", "customer", "carrier", "service_level", "shipping_address", "shipped_by", "invoice", "pick"},
		}),
	})
}
