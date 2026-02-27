package apikeyep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to rotate an API key, optionally overriding the expiration
type RotateAPIKeyRequest struct {
	// The unique identifier for the API key to rotate.
	APIKeyID string `path:"id"`
	// Optional expiration time override for the new API key.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (*RotateAPIKeyRequest) SchemaExample() any {
	return map[string]any{
		"expires_at": "2026-12-31T23:59:59Z",
	}
}

const rotateAPIKeyEndpointDescription string = `This endpoint rotates an API key by revoking the existing key and creating a new
replacement with the same name, role, and owner. The new key inherits the old key's expiration unless an explicit expires_at override is provided.
The new API key secret is returned once and is not retrievable after creation.`

type RotateAPIKeyEndpoint struct{}

func (e *RotateAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RotateAPIKeyRequest, *apiresource.CreatedAPIKey] {
	return &apiendpoint.APIEndpoint[*RotateAPIKeyRequest, *apiresource.CreatedAPIKey]{
		Title:             "Rotate API Key",
		Description:       rotateAPIKeyEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/api-keys/{id}/actions/rotate",
		ContentType:       "application/json",
		Request:           &RotateAPIKeyRequest{},
		Response:          &apiresource.CreatedAPIKey{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RotateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError) {
			return svc.(APIKeySvc).RotateAPIKey
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
