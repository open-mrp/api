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
	// Filter API keys by status. Valid values: active, expired, revoked.
	Status []constants.APIKeyStatus `query:"status" default:"active,expired,revoked"`
}

const listAPIKeysEndpointDescription string = `This endpoint returns a paginated list of API keys for the target account.
Supports cursor-based pagination and optional search filtering by name.`

type ListAPIKeysEndpoint struct{}

func (e *ListAPIKeysEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAPIKeysRequest, *apiresource.List[apiresource.APIKey]] {
	return &apiendpoint.APIEndpoint[*ListAPIKeysRequest, *apiresource.List[apiresource.APIKey]]{
		Title:             "List API Keys",
		Description:       listAPIKeysEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/auth/api-keys",
		Request:           &ListAPIKeysRequest{},
		Response:          &apiresource.List[apiresource.APIKey]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAPIKeysRequest) (*apiresource.List[apiresource.APIKey], *apierror.APIError) {
			return svc.(APIKeySvc).ListAPIKeys
		},
	}
}
