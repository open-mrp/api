package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveRoleRequest is a request to retrieve a role by ID.
type RetrieveRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
}

// Returns a role by ID, including its permissions.
type RetrieveRoleEndpoint struct{}

func (e *RetrieveRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRoleRequest, *apiresource.Role] {
	return (&apiendpoint.APIEndpoint[*RetrieveRoleRequest, *apiresource.Role]{
		Title:             "Retrieve Role",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRoleRequest) (*apiresource.Role, *apierror.APIError) {
			return svc.(RoleSvc).GetRole
		},
		ObjectType: constants.ObjectTypeRole,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "owner.account", "permissions"},
		}),
	})
}
