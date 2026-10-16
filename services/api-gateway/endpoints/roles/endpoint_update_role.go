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
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update a role.
type UpdateRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
	// The role's name, unique within your account.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Access levels to change. Areas left out keep their level.
	Access *RoleAccessInput `json:"access,omitzero"`
	// Permission codes to hold beyond the access levels. Replaces the role's grants.
	Grants field.Optional[[]string] `json:"grants,omitzero"`
	// Permission codes the access levels include that the role shouldn't hold. Replaces the role's revokes.
	Revokes field.Optional[[]string] `json:"revokes,omitzero"`
	// The role's whole permission list, replacing its access, grants and revokes. Kept for older clients.
	Permissions field.Optional[[]string] `json:"permissions,omitzero" visibility:"internal"`
}

var sampleUpdateRoleRequest = &UpdateRoleRequest{
	Access: &RoleAccessInput{Sales: field.Some(constants.AccessLevelEdit)},
	Grants: field.Some([]string{"invoices:read"}),
}

func (*UpdateRoleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateRoleRequest)
}

// Updates a role.
//
// Only fields you send are changed, including each area inside `access`. Changes apply to everyone with the role from their next request. You can only give a role permissions you hold yourself, unless you're an admin. The system roles OpenMRP owns can't be changed.
type UpdateRoleEndpoint struct{}

func (e *UpdateRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateRoleRequest, *apiresource.Role] {
	return (&apiendpoint.APIEndpoint[*UpdateRoleRequest, *apiresource.Role]{
		Title:             "Update Role",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).UpdateRole
		},
		ObjectType:    constants.ObjectTypeRole,
		IncludeConfig: roleIncludes(),
	})
}
