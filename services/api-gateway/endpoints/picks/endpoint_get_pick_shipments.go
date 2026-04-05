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
	// The ID of the pick.
	PickID string  `path:"id" validate:"required"`
	Query  *string `query:"q"`
	// The maximum number of results to return.
	Limit *int32 `query:"limit"`
	// The number of results to skip.
	Offset *int32 `query:"offset"`
}

type GetPickShipmentsEndpoint struct{}

func (e *GetPickShipmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPickShipmentsRequest, *apiresource.PickShipmentsResponse] {
	return &apiendpoint.APIEndpoint[*GetPickShipmentsRequest, *apiresource.PickShipmentsResponse]{
		Title:             "Get Pick Shipments",
		Description:       "Returns the shipment numbers associated with a pick.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/picks/{id}/shipments",
		Request:           &GetPickShipmentsRequest{},
		Response:          &apiresource.PickShipmentsResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPickShipmentsRequest) (*apiresource.PickShipmentsResponse, *apierror.APIError) {
			return svc.(PickSvc).GetPickShipments
		},
	}
}
