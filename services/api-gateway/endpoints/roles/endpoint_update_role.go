package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// UpdateRoleRequest is a request to update a role.
type UpdateRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
	// New display name for the role, unique within the account.
	//
	// Omit to leave unchanged.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Full replacement set of permissions, in `{domain}:{action}` format, such as `customers:read`.
	//
	// Replaces all existing permissions on the role. Pass an empty array to remove all permissions, or omit to leave them unchanged.
	Permissions field.Optional[[]string] `json:"permissions,omitzero"`
}

var sampleUpdateRoleName = "Updated Manager"
var sampleUpdateRolePerms = []string{"customers:read", "customers:update"}
var sampleUpdateRoleRequest = &UpdateRoleRequest{
	Name:        field.Some(sampleUpdateRoleName),
	Permissions: field.Some(sampleUpdateRolePerms),
}

func (*UpdateRoleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateRoleRequest)
}

// Partially updates a custom role's name or permissions.
//
// Provided permissions replace all existing ones; global roles cannot be modified.
type UpdateRoleEndpoint struct{}

func (e *UpdateRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateRoleRequest, *apiresource.Role] {
	return (&apiendpoint.APIEndpoint[*UpdateRoleRequest, *apiresource.Role]{
		Title:               "Update Role",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/identity/roles/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).UpdateRole
		},
		ObjectType: constants.ObjectTypeRole,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "owner.account", "permissions"},
		}),
	})
}
