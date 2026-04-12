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

// CreateRoleRequest is the request to create a new role.
type CreateRoleRequest struct {
	// The display name of the role.
	Name string `json:"name" validate:"required,max=255"`
	// The permissions to attach to this role in "domain:action" format.
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

type CreateRoleEndpoint struct{}

func (e *CreateRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateRoleRequest, *apiresource.Role] {
	return &apiendpoint.APIEndpoint[*CreateRoleRequest, *apiresource.Role]{
		Title:             "Create Role",
		Description:       "Creates a new custom role with the specified permissions.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles",
		Request:           &CreateRoleRequest{},
		Response:          &apiresource.Role{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).CreateRole
		},
		LocationFunc: func(resp *apiresource.Role) string {
			return "/v1/identity/roles/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "permissions"},
		}),
	}
}
