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

// UpdateRoleRequest is a request to update a role.
type UpdateRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name" nullable:"false" validate:"omitempty,max=255"`
	// Permissions in `<domain>:<action>` format. Replaces all existing permissions; omit to leave unchanged.
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

// Partially updates a custom role's name or permissions. Provided permissions replace all existing ones; global roles cannot be modified.
type UpdateRoleEndpoint struct{}

func (e *UpdateRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateRoleRequest, *apiresource.Role] {
	return (&apiendpoint.APIEndpoint[*UpdateRoleRequest, *apiresource.Role]{
		Title:             "Update Role",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles/{id}",
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
	})
}
