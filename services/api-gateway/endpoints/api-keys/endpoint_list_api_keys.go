package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListAPIKeysRequest embeds PaginationRequest and adds a status filter.
type ListAPIKeysRequest struct {
	apiresource.PaginationRequest
	// Filter API keys by status.
	Status []constants.APIKeyStatus `query:"status" default:"active,expired,revoked"`
}

type ListAPIKeysEndpoint struct{}

func (e *ListAPIKeysEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAPIKeysRequest, *apiresource.List[apiresource.APIKey]] {
	return &apiendpoint.APIEndpoint[*ListAPIKeysRequest, *apiresource.List[apiresource.APIKey]]{
		Title:             "List API Keys",
		Description:       "Returns a paginated list of API keys for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/auth/api-keys",
		Request:           &ListAPIKeysRequest{},
		Response:          &apiresource.List[apiresource.APIKey]{},
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
	}
}
