package roleep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a role.
type DeleteRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
}

// Deletes a role along with the permissions granted through it.
//
// Only roles owned by your account can be deleted; the system-owned roles shared across all accounts cannot. A role that is still assigned to at least one user is rejected, so move those users to another role first.
type DeleteRoleEndpoint struct{}

func (e *DeleteRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteRoleRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteRoleRequest, *object.Deleted]{
		Title:             "Delete Role",
		Method:            http.MethodDelete,
		Route:             "/v1/identity/roles/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteRoleRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(RoleSvc).DeleteRole, constants.ObjectTypeRole, "id")
		},
	})
}
