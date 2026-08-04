package apikeyep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to rotate an API key.
type RotateAPIKeyRequest struct {
	// ID of the API key to rotate.
	//
	// The key must not already be revoked.
	APIKeyID string `path:"id" validate:"required"`
	// When the replacement key should expire.
	//
	// If omitted, the replacement inherits the expiration of the key being rotated.
	ExpiresAt field.Optional[time.Time] `json:"expires_at,omitzero"`
	// When the old key should stop authenticating requests.
	//
	// If omitted, the old key is revoked immediately. Set a future timestamp — up to 30 days out — to keep the old key working during a migration window; a timestamp in the past revokes it immediately.
	RevokeAt field.Optional[time.Time] `json:"revoke_at,omitzero" validate:"omitempty,max_days_ahead=30"`
}

func (*RotateAPIKeyRequest) SchemaExample() any {
	return map[string]any{
		"expires_at": "2026-12-31T23:59:59Z",
		"revoke_at":  "2026-06-16T00:00:00Z",
	}
}

// Rotates an [API key](https://docs.augno.com/api/api-keys) by revoking the existing key and issuing a replacement with the same name, role, and expiration (unless overridden).
//
// The replacement is a new key with its own ID; the rotated key keeps its ID and stays in the list, moving to a `revoked` status once its revocation takes effect. Use `revoke_at` to keep the old key working while you roll the new secret out.
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
		AgentTool:         false,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeCreatedAPIKey,
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
