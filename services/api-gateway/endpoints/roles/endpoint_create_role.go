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
)

// CreateRoleRequest is a request to create a role.
type CreateRoleRequest struct {
	// Display name for the role, unique within the account.
	Name string `json:"name" validate:"required,max=255"`
	// Permissions to grant, in `{domain}:{action}` format, such as `customers:read`.
	//
	// The action must be one of `create`, `read`, `update`, or `delete`. Omit to create a role with no permissions.
	Permissions []string `json:"permissions,omitzero"`
}

var sampleCreateRoleRequest = &CreateRoleRequest{
	Name: "Warehouse Manager",
	Permissions: []string{
		"customers:create", "customers:read", "customers:update",
		"invoices:read",
	},
}

func (*CreateRoleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateRoleRequest)
}

// Creates a custom role with the specified permissions.
//
// Roles created through the API always have type `user`.
type CreateRoleEndpoint struct{}

func (e *CreateRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateRoleRequest, *apiresource.Role] {
	return (&apiendpoint.APIEndpoint[*CreateRoleRequest, *apiresource.Role]{
		Title:               "Create Role",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/identity/roles",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionCreate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).CreateRole
		},
		LocationFunc: func(resp *apiresource.Role) string {
			return "/v1/identity/roles/" + resp.ID
		},
		ObjectType: constants.ObjectTypeRole,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "owner.account", "permissions"},
		}),
	})
}
