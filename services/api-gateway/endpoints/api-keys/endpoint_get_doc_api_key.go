package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Returns a sandbox API key for documentation. Reuses an existing valid key or creates one if none exists.
type GetDocAPIKeyEndpoint struct{}

func (e *GetDocAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.CreatedAPIKey] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.CreatedAPIKey]{
		Title:             "Get Documentation API Key",
		Method:            http.MethodPost,
		Route:             "/v1/auth/api-keys/actions/fetch-doc-api-key",
		ContentType:       "application/json",
		Request:           &apiresource.EmptyResource{},
		Response:          &apiresource.CreatedAPIKey{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
		Extras: apiendpoint.APIEndpointExtras{
			ShieldResponseBody: true,
		},
	}).WithDocSource(e)
}
