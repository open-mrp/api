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
	// Opaque token that ties this lookup to the autocomplete requests the suggestion came from.
	//
	// Pass the same token used for those autocomplete requests so the whole address entry is treated as one lookup.
	SessionToken *string `query:"session_token"` // #nosec G117 -- not a secret, Google Maps session correlation token
}

// Returns the full parsed address for a suggestion returned by address autocomplete.
//
// Use this after the user picks a suggestion to get the street, city, state, postal code, and country to prefill an address form. Nothing is saved by this lookup; create an address separately to keep it.
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
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAddressDetailsRequest) (*apiresource.AddressDetailsResult, *apierror.APIError) {
			return svc.(AddressValidationSvc).GetAddressDetails
		},
	})
}
