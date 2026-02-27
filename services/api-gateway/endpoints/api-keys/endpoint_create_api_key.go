package apikeyep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to create an API key
type CreateAPIKeyRequest struct {
	// The role ID for the API key.
	RoleID string `json:"role_id" validate:"required"`
	// The name for the API key.
	Name string `json:"name" validate:"required"`
	// Optional expiration time for the API key.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

var sampleCreateAPIKeyRequest = &CreateAPIKeyRequest{
	RoleID: apiresource.SampleRoleID,
	Name:   apiresource.SampleAPIKeyName,
}

func (*CreateAPIKeyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAPIKeyRequest)
}

const createAPIKeyEndpointDescription string = `This endpoint is used to create an API key. Once completed, the API key object is
returned, and the API key secret is returned. The secret is only returned once at creation, and is not retrievable after creation.`

type CreateAPIKeyEndpoint struct{}

func (e *CreateAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAPIKeyRequest, *apiresource.CreatedAPIKey] {
	return &apiendpoint.APIEndpoint[*CreateAPIKeyRequest, *apiresource.CreatedAPIKey]{
		Title:             "Create API Key",
		Description:       createAPIKeyEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/api-keys",
		ContentType:       "application/json",
		Request:           &CreateAPIKeyRequest{},
		Response:          &apiresource.CreatedAPIKey{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError) {
			return svc.(APIKeySvc).CreateAPIKey
		},
		LocationFunc: func(resp *apiresource.CreatedAPIKey) string {
			return "/v1/auth/api-keys/" + resp.APIKeyInfo.ID
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
			ShieldResponseBody:     true,
		},
	}
}
