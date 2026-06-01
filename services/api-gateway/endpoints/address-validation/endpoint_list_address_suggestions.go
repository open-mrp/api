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
	// Autocomplete input text.
	Input string `query:"input" validate:"required"`
	// Session token for grouping autocomplete requests.
	SessionToken *string `query:"session_token"` // #nosec G117 -- not a secret, Google Maps session correlation token
}

// Returns address suggestions based on input text.
type AutocompleteAddressEndpoint struct{}

func (e *AutocompleteAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*AutocompleteAddressRequest, *apiresource.List[apiresource.AddressSuggestion]] {
	return (&apiendpoint.APIEndpoint[*AutocompleteAddressRequest, *apiresource.List[apiresource.AddressSuggestion]]{
		Title:             "List Address Suggestions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/addresses/suggestions",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAddressSuggestion,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AutocompleteAddressRequest) (*apiresource.List[apiresource.AddressSuggestion], *apierror.APIError) {
			return svc.(AddressValidationSvc).AutocompleteAddress
		},
	})
}
