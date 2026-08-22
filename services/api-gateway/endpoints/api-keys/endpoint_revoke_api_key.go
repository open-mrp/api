package apikeyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to revoke an API key.
type RevokeAPIKeyRequest struct {
	// API key ID.
	APIKeyID string `path:"id" validate:"required"`
}

// Revokes an [API key](https://docs.openmrp.ai/api/api-keys).
//
// Revocation takes effect immediately and cannot be undone; any request still presenting the key is rejected. The key record is kept, so it stays visible in the key list with a `revoked` status. To replace a key without an interruption in access, use Rotate API Key instead.
type RevokeAPIKeyEndpoint struct{}

func (e *RevokeAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *apiresource.EmptyResource]{
		Title:             "Revoke API Key",
		Method:            http.MethodDelete,
		Route:             "/v1/auth/api-keys/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RevokeAPIKeyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(APIKeySvc).RevokeAPIKey
		},
	})
}
