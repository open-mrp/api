package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteRoleRequest is a request to delete a role.
type DeleteRoleRequest struct {
	// Role ID.
	RoleID string `path:"id" validate:"required"`
}

type DeleteRoleEndpoint struct{}

func (e *DeleteRoleEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteRoleRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteRoleRequest, *apiresource.EmptyResource]{
		Title:             "Delete Role",
		Description:       "Deletes an account-owned role and its associated permissions. Global roles cannot be deleted.",
		Method:            http.MethodDelete,
		Route:             "/v1/identity/roles/{id}",
		ContentType:       "application/json",
		Request:           &DeleteRoleRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteRoleRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(RoleSvc).DeleteRole
		},
	}
}
