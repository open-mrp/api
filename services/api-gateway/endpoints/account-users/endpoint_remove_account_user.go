package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove an account user.
type RemoveAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
}

// Removes a user from the target account.
type RemoveAccountUserEndpoint struct{}

func (e *RemoveAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveAccountUserRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveAccountUserRequest, *apiresource.EmptyResource]{
		Title:             "Remove Account User",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}/actions/remove",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).RemoveAccountUser
		},
	})
}
