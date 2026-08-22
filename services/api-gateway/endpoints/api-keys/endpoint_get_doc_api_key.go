package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Returns the sandbox API key used to try requests from the API documentation.
//
// Reuses the existing documentation key if it is still valid, rotates it if it has expired, and creates one if none exists. If the key was explicitly revoked, returns an error instead of regenerating it, on the assumption that the revocation was deliberate.
//
// The caller must be signed in as a member of the account they are targeting, and that account must be in sandbox mode; API-key authentication and production accounts are rejected.
type GetDocAPIKeyEndpoint struct{}

func (e *GetDocAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.CreatedAPIKey] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.CreatedAPIKey]{
		Title:             "Get Documentation API Key",
		Method:            http.MethodPost,
		Route:             "/v1/auth/api-keys/actions/fetch-doc-api-key",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ObjectType:        constants.ObjectTypeCreatedAPIKey,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.CreatedAPIKey, *apierror.APIError) {
			return func(ctx context.Context, _ *apiresource.EmptyResource) (*apiresource.CreatedAPIKey, *apierror.APIError) {
				return svc.(APIKeySvc).GetOrCreateDocAPIKey(ctx)
			}
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{"role", "role.permissions"},
			PathPrefix: "api_key_info",
		}),
	})
}
