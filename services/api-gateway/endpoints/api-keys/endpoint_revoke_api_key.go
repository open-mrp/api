package apikeyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *RevokeAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *object.Deleted]{
		Title:             "Revoke API Key",
		Method:            http.MethodDelete,
		Route:             "/v1/auth/api-keys/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeAPIKey,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{RoleType: constants.RoleTypeAdmin},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RevokeAPIKeyRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(APIKeySvc).RevokeAPIKey, constants.ObjectTypeAPIKey, "id")
		},
	})
}
