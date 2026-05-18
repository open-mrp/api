package apikeyep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to rotate an API key.
type RotateAPIKeyRequest struct {
	// API key ID to rotate.
	APIKeyID string `path:"id" validate:"required"`
	// Expiration timestamp override. If omitted, the previous key's expiration is used.
	ExpiresAt *time.Time `json:"expires_at,omitempty" nullable:"false"`
}

func (*RotateAPIKeyRequest) SchemaExample() any {
	return map[string]any{
		"expires_at": "2026-12-31T23:59:59Z",
	}
}

// Rotates an [API key](https://docs.augno.com/api/api-keys) by revoking the existing key and issuing a replacement with the same name, role, and expiration (unless overridden).
//
// The secret key is returned once and cannot be retrieved later, so you should store it securely. We provide some [recommendations](https://docs.augno.com/api/managing-api-keys) on how you can manage your API keys.
type RotateAPIKeyEndpoint struct{}

func (e *RotateAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RotateAPIKeyRequest, *apiresource.CreatedAPIKey] {
	return (&apiendpoint.APIEndpoint[*RotateAPIKeyRequest, *apiresource.CreatedAPIKey]{
		Title:             "Rotate API Key",
		Method:            http.MethodPost,
		Route:             "/v1/auth/api-keys/{id}/actions/rotate",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RotateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError) {
			return svc.(APIKeySvc).RotateAPIKey
		},
		LocationFunc: func(resp *apiresource.CreatedAPIKey) string {
			return "/v1/auth/api-keys/" + resp.APIKeyInfo.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{"role", "role.permissions"},
			PathPrefix: "api_key_info",
		}),
	})
}
