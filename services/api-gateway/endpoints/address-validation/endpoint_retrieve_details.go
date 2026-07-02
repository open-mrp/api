package addressvalidationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get details for a place.
type RetrieveAddressDetailsRequest struct {
	// The place ID to look up, as returned in the `id` field of an address suggestion.
	PlaceID string `path:"id" validate:"required"`
	// Session token for grouping with a previous autocomplete request.
	SessionToken *string `query:"session_token"` // #nosec G117 -- not a secret, Google Maps session correlation token
}

// Returns the full parsed address for a place returned by address autocomplete.
type RetrieveAddressDetailsEndpoint struct{}

func (e *RetrieveAddressDetailsEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAddressDetailsRequest, *apiresource.AddressDetailsResult] {
	return (&apiendpoint.APIEndpoint[*RetrieveAddressDetailsRequest, *apiresource.AddressDetailsResult]{
		Title:             "Retrieve Address Details",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/addresses/details/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAddressDetailsResult,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAddressDetailsRequest) (*apiresource.AddressDetailsResult, *apierror.APIError) {
			return svc.(AddressValidationSvc).GetAddressDetails
		},
	})
}
