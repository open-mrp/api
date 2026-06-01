package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// CreateRoleRequest is a request to create a role.
type CreateRoleRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Permissions to attach in `<domain>:<action>` format.
	Permissions []string `json:"permissions"`
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

// Creates a new role with the specified permissions.
type CreateRoleEndpoint struct{}

func (e *CreateRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateRoleRequest, *apiresource.Role] {
	return (&apiendpoint.APIEndpoint[*CreateRoleRequest, *apiresource.Role]{
		Title:             "Create Role",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
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
