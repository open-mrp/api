package userep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
//
// Users may always retrieve their own profile. Retrieving another user requires permission to read team users, and only users who belong or belonged to the account you are acting in are found: a member since removed is still found, as records such as a customer's sales rep keep naming them, but their profile can no longer be updated.
type RetrieveUserEndpoint struct{}

func (e *RetrieveUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveUserRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*RetrieveUserRequest, *apiresource.User]{
		Title:             "Retrieve User",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/users/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUser,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainTeamUsers, Action: types.ActionRead},
		}, SelfPathParam: "id"},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveUserRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(UserSvc).GetUser
		},
	})
}
