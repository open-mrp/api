package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetRoleRequest is the request to retrieve a single role by ID.
type GetRoleRequest struct {
	// The ID of the role to retrieve.
	RoleID string `path:"id" validate:"required"`
}

type GetRoleEndpoint struct{}

func (e *GetRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetRoleRequest, *apiresource.Role] {
	return &apiendpoint.APIEndpoint[*GetRoleRequest, *apiresource.Role]{
		Title:             "Get Role",
		Description:       "Returns a single role by its ID, including its structured permissions.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles/{id}",
		Request:           &GetRoleRequest{},
		Response:          &apiresource.Role{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).GetRole
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "owner.account", "permissions"},
		}),
	}
}
