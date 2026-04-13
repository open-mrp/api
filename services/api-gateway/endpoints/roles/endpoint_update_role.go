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

// UpdateRoleRequest is the request to update a role.
type UpdateRoleRequest struct {
	// The ID of the role to update.
	RoleID string `path:"id" validate:"required"`
	// The new display name for the role.
	Name *string `json:"name" nullable:"false" validate:"omitempty,max=255"`
	// The full set of permissions to replace existing ones with in `<domain>:<action>` format. If omitted, permissions are not changed.
	Permissions *[]string `json:"permissions" nullable:"false"`
}

var sampleUpdateRoleName = "Updated Manager"
var sampleUpdateRolePerms = []string{"customers:read", "customers:update"}
var sampleUpdateRoleRequest = &UpdateRoleRequest{
	Name:        &sampleUpdateRoleName,
	Permissions: &sampleUpdateRolePerms,
}

func (*UpdateRoleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateRoleRequest)
}

type UpdateRoleEndpoint struct{}

func (e *UpdateRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateRoleRequest, *apiresource.Role] {
	return &apiendpoint.APIEndpoint[*UpdateRoleRequest, *apiresource.Role]{
		Title:             "Update Role",
		Description:       "Partially updates a custom role's name or permissions. Provided permissions replace all existing ones; global roles cannot be modified.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles/{id}",
		Request:           &UpdateRoleRequest{},
		Response:          &apiresource.Role{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).UpdateRole
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "owner.account", "permissions"},
		}),
	}
}
