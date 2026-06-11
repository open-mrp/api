package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to revoke an API key.
type RevokeAPIKeyRequest struct {
	// API key ID.
	APIKeyID string `path:"id" validate:"required"`
}

// Revokes an [API key](https://docs.augno.com/api/api-keys).
//
// Revocation takes effect immediately and cannot be undone; revoked keys can no longer be used to authenticate requests. To replace a key without losing access, use Rotate API Key instead.
type RevokeAPIKeyEndpoint struct{}

func (e *RevokeAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *apiresource.EmptyResource]{
		Title:             "Revoke API Key",
		Method:            http.MethodDelete,
		Route:             "/v1/auth/api-keys/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RevokeAPIKeyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(APIKeySvc).RevokeAPIKey
		},
	})
}
