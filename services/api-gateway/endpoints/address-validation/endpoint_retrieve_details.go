package addressvalidationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get details for a place.
type RetrieveAddressDetailsRequest struct {
	// Google Places ID.
	PlaceID string `path:"id" validate:"required"`
	// Session token for grouping with a previous autocomplete request.
	SessionToken *string `query:"session_token"` // #nosec G117 -- not a secret, Google Maps session correlation token
}

type RetrieveAddressDetailsEndpoint struct{}

func (e *RetrieveAddressDetailsEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAddressDetailsRequest, *apiresource.AddressDetailsResult] {
	return &apiendpoint.APIEndpoint[*RetrieveAddressDetailsRequest, *apiresource.AddressDetailsResult]{
		Title:             "Retrieve Address Details",
		Description:       "Returns parsed address components for a Google Places ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/addresses/details/{id}",
		Request:           &RetrieveAddressDetailsRequest{},
		Response:          &apiresource.AddressDetailsResult{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAddressDetailsRequest) (*apiresource.AddressDetailsResult, *apierror.APIError) {
			return svc.(AddressValidationSvc).GetAddressDetails
		},
	}
}
