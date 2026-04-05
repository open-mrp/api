package carrierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetCarrierRequest is the request to retrieve a single carrier by ID.
type GetCarrierRequest struct {
	// The ID of the carrier to retrieve.
	CarrierID string `path:"id" validate:"required"`
}

type GetCarrierEndpoint struct{}

func (e *GetCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetCarrierRequest, *apiresource.Carrier] {
	return &apiendpoint.APIEndpoint[*GetCarrierRequest, *apiresource.Carrier]{
		Title:             "Get Carrier",
		Description:       "Returns a single carrier by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/carriers/{id}",
		Request:           &GetCarrierRequest{},
		Response:          &apiresource.Carrier{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).GetCarrier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: "carrier",
			Fields:     []string{"owner", "service_levels"},
		}),
	}
}
