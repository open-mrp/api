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

// Request to update a role.
type UpdateRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
	// New display name for the role.
	//
	// Returns a conflict error if another role in your account already uses this name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Full replacement set of permissions, in `{permission}:{action}` format, such as `customers:read`.
	//
	// The role's existing permissions are discarded and replaced with exactly what you send, so include every permission the role should keep. Sending an empty array strips the role of all access, while leaving the field out keeps the current permissions untouched.
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

// Updates a role's name or the set of permissions it grants.
//
// Only roles owned by your account can be updated; the system-owned roles shared across all accounts are rejected. Permission changes apply to every user already assigned the role, starting with their next request.
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
