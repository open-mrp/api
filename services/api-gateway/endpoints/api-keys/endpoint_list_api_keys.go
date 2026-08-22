package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list API keys.
type ListAPIKeysRequest struct {
	apiresource.PaginationRequest
	// API key statuses to filter by.
	//
	// - `active`: the key still authenticates requests. A key whose revocation is scheduled for a future time is still active until that time arrives.
	// - `expired`: the key passed its expiration time without having been revoked.
	// - `revoked`: the key was revoked, which takes precedence over expiration.
	//
	// When omitted, keys of every status are returned.
	Statuses []constants.APIKeyStatus `query:"statuses" default:"active,expired,revoked"`
}

// Returns a paginated list of [API keys](https://docs.openmrp.ai/api/api-keys), newest first.
//
// Only keys belonging to the account making the request are returned. The search term matches against the key name.
type ListAPIKeysEndpoint struct{}

func (e *ListAPIKeysEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAPIKeysRequest, *apiresource.List[apiresource.APIKey]] {
	return (&apiendpoint.APIEndpoint[*ListAPIKeysRequest, *apiresource.List[apiresource.APIKey]]{
		Title:             "List API Keys",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/auth/api-keys",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAPIKey,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAPIKeysRequest) (*apiresource.List[apiresource.APIKey], *apierror.APIError) {
			return svc.(APIKeySvc).ListAPIKeys
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{"role", "role.permissions"},
		}),
	})
}
