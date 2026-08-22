package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a role.
type RetrieveRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
}

// Retrieves a single role by ID.
//
// Both the roles your account owns and the system-owned roles shared by every account can be retrieved.
type RetrieveRoleEndpoint struct{}

func (e *RetrieveRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRoleRequest, *apiresource.Role] {
	return (&apiendpoint.APIEndpoint[*RetrieveRoleRequest, *apiresource.Role]{
		Title:               "Retrieve Role",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/identity/roles/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).GetRole
		},
		ObjectType: constants.ObjectTypeRole,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "owner.account", "permissions"},
		}),
	})
}
