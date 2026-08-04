package addressvalidationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request for address autocomplete.
type AutocompleteAddressRequest struct {
	// Partial address text to generate suggestions for.
	Input string `query:"input" validate:"required"`
	// Opaque token that groups a series of related autocomplete requests into a single session.
	//
	// Reuse the same token for each keystroke of one address entry, and again when you retrieve the details of the suggestion the user picks, so the whole entry is treated as one lookup.
	SessionToken *string `query:"session_token"` // #nosec G117 -- not a secret, Google Maps session correlation token
}

// Returns address suggestions for partial address text, for use in type-ahead address entry.
//
// Only street addresses are suggested; cities, regions, and business listings are not returned. Suggestions are lookup results, not saved addresses in your account. Pass a suggestion's `id` to the address details endpoint to get the full parsed address.
type AutocompleteAddressEndpoint struct{}

func (e *AutocompleteAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*AutocompleteAddressRequest, *apiresource.List[apiresource.AddressSuggestion]] {
	return (&apiendpoint.APIEndpoint[*AutocompleteAddressRequest, *apiresource.List[apiresource.AddressSuggestion]]{
		Title:             "List Address Suggestions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/addresses/suggestions",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAddressSuggestion,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AutocompleteAddressRequest) (*apiresource.List[apiresource.AddressSuggestion], *apierror.APIError) {
			return svc.(AddressValidationSvc).AutocompleteAddress
		},
	})
}
