package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list API keys.
type ListAPIKeysRequest struct {
	apiresource.PaginationRequest
	// API key statuses to filter by.
	Statuses []constants.APIKeyStatus `query:"statuses" default:"active,expired,revoked"`
}

// Returns a paginated list of [API keys](https://docs.augno.com/api/api-keys).
type ListAPIKeysEndpoint struct{}

func (e *ListAPIKeysEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAPIKeysRequest, *apiresource.List[apiresource.APIKey]] {
	return (&apiendpoint.APIEndpoint[*ListAPIKeysRequest, *apiresource.List[apiresource.APIKey]]{
		Title:             "List API Keys",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/auth/api-keys",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAPIKeysRequest) (*apiresource.List[apiresource.APIKey], *apierror.APIError) {
			return svc.(APIKeySvc).ListAPIKeys
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{"role", "role.permissions"},
		}),
	})
}
