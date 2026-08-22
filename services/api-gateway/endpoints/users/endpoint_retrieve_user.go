package userep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a user.
type RetrieveUserRequest struct {
	// Identifier for the user.
	//
	// Accepts the user's ID, email address, or username, tried in that order.
	UserID string `path:"id" validate:"required"`
}

// Retrieves a user's global profile.
//
// The profile is shared across every account the user belongs to; account-specific details such as their status, role, and department live on the account user record instead.
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
