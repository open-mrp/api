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

func (e *RetrieveShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShipmentRequest, *apiresource.Shipment] {
	return (&apiendpoint.APIEndpoint[*RetrieveShipmentRequest, *apiresource.Shipment]{
		Title:             "Retrieve Shipment",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeShipment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
			return svc.(ShipmentSvc).GetShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "sales_order", "customer", "freight", "shipping_address", "shipped_by", "shipped_by.user", "invoice", "pick"},
		}),
	})
}
