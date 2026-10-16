package apikeyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"

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

// Revokes an [API key](https://docs.openmrp.ai/api/api-keys). Internal: 1.0.forge-preview.9 callers only; forge revokes
// with POST /v1/auth/api-keys/{id}/actions/revoke.
type RevokeAPIKeyEndpoint struct{}

func (e *RevokeAPIKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*RevokeAPIKeyRequest, *object.Deleted]{
		Title:             "Revoke API Key",
		Method:            http.MethodDelete,
		Route:             "/v1/auth/api-keys/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeAPIKey,
		Public:            false,
		AgentTool:         false,
		Auth:              authz.Policy{RoleType: constants.RoleTypeAdmin},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RevokeAPIKeyRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(APIKeySvc).RevokeAPIKey, constants.ObjectTypeAPIKey, "id")
		},
	})
}

// Revokes an [API key](https://docs.openmrp.ai/api/api-keys), moving it to `revoked`.
//
// Revocation takes effect immediately and can't be undone; requests still presenting the key are rejected. The key stays in the list. To replace a key without a gap in access, rotate it instead.
type RevokeAPIKeyActionEndpoint struct{}

func (*RevokeAPIKeyActionEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.APIKey] {
	e := lifecycle.Action("Revoke API Key", "/v1/auth/api-keys/{id}/actions/revoke", constants.ObjectTypeAPIKey,
		authz.Policy{RoleType: constants.RoleTypeAdmin},
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.APIKey] {
			return svc.(APIKeySvc).RevokeAPIKeyAction
		})
	e.IncludeConfig = apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeAPIKey,
		Fields:     []string{"role", "role.permissions"},
	})
	return e
}
