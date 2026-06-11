package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetPickShipmentsRequest is the request to get shipments for a pick.
type GetPickShipmentsRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
	// Search query that filters shipment numbers by substring match.
	Query *string `query:"q"`
	// Maximum number of results to return.
	Limit *int32 `query:"limit"`
	// Number of results to skip.
	Offset *int32 `query:"offset"`
}

// Returns the shipment numbers associated with a pick.
//
// Shipments are matched through the pick's sales order, so the list covers every shipment created for that order.
type GetPickShipmentsEndpoint struct{}

func (e *GetPickShipmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPickShipmentsRequest, *apiresource.PickShipmentsResponse] {
	return (&apiendpoint.APIEndpoint[*GetPickShipmentsRequest, *apiresource.PickShipmentsResponse]{
		Title:             "Get Pick Shipments",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}/shipments",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPickShipmentsRequest) (*apiresource.PickShipmentsResponse, *apierror.APIError) {
			return svc.(PickSvc).GetPickShipments
		},
	})
}
