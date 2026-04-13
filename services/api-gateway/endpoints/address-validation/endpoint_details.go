package addressvalidationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetAddressDetailsRequest is the request to get details for a place.
type GetAddressDetailsRequest struct {
	// The Google Places ID to look up.
	PlaceID string `path:"id" validate:"required"`
	// An optional session token for grouping with a previous autocomplete request.
	SessionToken *string `query:"session_token"` // #nosec G117 -- not a secret, Google Maps session correlation token
}

type GetAddressDetailsEndpoint struct{}

func (e *GetAddressDetailsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAddressDetailsRequest, *apiresource.AddressDetailsResult] {
	return &apiendpoint.APIEndpoint[*GetAddressDetailsRequest, *apiresource.AddressDetailsResult]{
		Title:             "Get Address Details",
		Description:       "Returns parsed address components for a Google Places ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/addresses/details/{id}",
		Request:           &GetAddressDetailsRequest{},
		Response:          &apiresource.AddressDetailsResult{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAddressDetailsRequest) (*apiresource.AddressDetailsResult, *apierror.APIError) {
			return svc.(AddressValidationSvc).GetAddressDetails
		},
	}
}
