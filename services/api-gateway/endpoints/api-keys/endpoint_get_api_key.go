package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
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
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
