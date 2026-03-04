package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetAPIKeyRequest is the request to retrieve a single API key by ID.
type GetAPIKeyRequest struct {
	// The ID of the API key to retrieve.
	APIKeyID string `path:"id"`
}

const getAPIKeyEndpointDescription string = `This endpoint returns a single API key's metadata by its ID.`

type GetAPIKeyEndpoint struct{}

func (e *GetAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAPIKeyRequest, *apiresource.APIKey] {
	return &apiendpoint.APIEndpoint[*GetAPIKeyRequest, *apiresource.APIKey]{
		Title:             "Get API Key",
		Description:       getAPIKeyEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/auth/api-keys/{id}",
		Request:           &GetAPIKeyRequest{},
		Response:          &apiresource.APIKey{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAPIKeyRequest) (*apiresource.APIKey, *apierror.APIError) {
			return svc.(APIKeySvc).GetAPIKey
		},
		IncludeConfig: &apiendpoint.IncludeConfig{
			Fields: []apiendpoint.IncludeField{
				{Key: "role", ObjectType: constants.ObjectTypeRole, JSONPaths: []string{"role"}},
			},
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
