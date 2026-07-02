package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteRoleRequest is a request to delete a role.
type DeleteRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
}

// Deletes a role and its associated permissions.
//
// Global roles and roles currently assigned to one or more users cannot be deleted.
type DeleteRoleEndpoint struct{}

func (e *DeleteRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteRoleRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteRoleRequest, *apiresource.EmptyResource]{
		Title:               "Delete Role",
		Method:              http.MethodDelete,
		Route:               "/v1/identity/roles/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteRoleRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(RoleSvc).DeleteRole
		},
	})
}
