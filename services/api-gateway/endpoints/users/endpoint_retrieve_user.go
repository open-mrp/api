package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a user by ID.
type RetrieveUserRequest struct {
	// User ID.
	UserID string `path:"id" validate:"required"`
}

// Returns a user by ID.
type RetrieveUserEndpoint struct{}

func (e *RetrieveUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveUserRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*RetrieveUserRequest, *apiresource.User]{
		Title:             "Retrieve User",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/users/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUser,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTeamUsers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveUserRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(UserSvc).GetUser
		},
	})
}
