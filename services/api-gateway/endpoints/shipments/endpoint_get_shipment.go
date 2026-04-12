package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetShipmentRequest is the request to retrieve a single shipment by ID.
type GetShipmentRequest struct {
	// The ID of the shipment to retrieve.
	ShipmentID string `path:"id" validate:"required"`
	// The list of related resources to include.
	Includes []string `query:"include"`
}

type GetShipmentEndpoint struct{}

func (e *GetShipmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetShipmentRequest, *apiresource.ShipmentDetail] {
	return &apiendpoint.APIEndpoint[*GetShipmentRequest, *apiresource.ShipmentDetail]{
		Title:             "Get Shipment",
		Description:       "Returns a single shipment by its ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments/{id}",
		Request:           &GetShipmentRequest{},
		Response:          &apiresource.ShipmentDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
			return svc.(ShipmentSvc).GetShipment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShipment,
			Fields:     []string{"lines", "shipping_cases", "sales_order", "customer", "carrier", "service_level", "shipping_address", "shipped_by", "invoice", "pick"},
		}),
	}
}
