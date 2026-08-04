package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove an account user.
type RemoveAccountUserRequest struct {
	// ID of the account user to remove.
	AccountUserID string `path:"id" validate:"required"`
}

// Removes a user from the account you are acting in.
//
// Removal is a soft delete: removed users are excluded from listings unless requested via `removed_scope`, they free the seat they occupied, and they can be restored with the activate action. Removing an already-removed user is a no-op. The user's profile itself is untouched, so their access to any other account they belong to is unaffected.
type RemoveAccountUserEndpoint struct{}

func (e *RemoveAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveAccountUserRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveAccountUserRequest, *apiresource.EmptyResource]{
		Title:               "Remove Account User",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users/{id}/actions/remove",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionDelete}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).RemoveAccountUser
		},
	})
}
