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

// Request to create a role.
type CreateRoleRequest struct {
	// The role's name, unique within your account.
	Name string `json:"name" validate:"required,max=255"`
	// A template to start from; its access and grants apply unless you set them. GET /v1/identity/role-templates lists them.
	Template field.Optional[constants.RoleTemplate] `json:"template,omitzero"`
	// Access levels per area. Areas left out take the template's level, or `none`.
	Access *RoleAccessInput `json:"access,omitzero"`
	// Permission codes to hold beyond the access levels. Replaces the template's grants when sent.
	Grants field.Optional[[]string] `json:"grants,omitzero"`
	// Permission codes the access levels include that the role shouldn't hold.
	Revokes []string `json:"revokes,omitzero"`
	// The role's whole permission list, instead of a template, access, grants and revokes. Kept for older clients.
	Permissions field.Optional[[]string] `json:"permissions,omitzero" visibility:"internal"`
}

var sampleCreateRoleRequest = &CreateRoleRequest{
	Name:     "Warehouse",
	Template: field.Some(constants.RoleTemplateWarehouse),
	Access:   &RoleAccessInput{Production: field.Some(constants.AccessLevelRead)},
	Revokes:  []string{"shipments:delete"},
}

func (*CreateRoleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateRoleRequest)
}

// Creates a role.
//
// Start from a `template`, then adjust `access`, `grants` and `revokes`. You can only give a role permissions you hold yourself, unless you're an admin. 409 if your account already has a role with the name.
type CreateRoleEndpoint struct{}

func (e *CreateRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateRoleRequest, *apiresource.Role] {
	return (&apiendpoint.APIEndpoint[*CreateRoleRequest, *apiresource.Role]{
		Title:             "Create Role",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionCreate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).CreateRole
		},
		LocationFunc: func(resp *apiresource.Role) string {
			return "/v1/identity/roles/" + resp.ID
		},
		ObjectType:    constants.ObjectTypeRole,
		IncludeConfig: roleIncludes(),
	})
}

func roleIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeRole,
		Fields:     []string{"owner", "owner.account", "permissions"},
	})
}
