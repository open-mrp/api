package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UnlockAccountUserRequest is the request to unlock an account user.
type UnlockAccountUserRequest struct {
	// The ID of the account user to unlock.
	AccountUserID string `path:"id" validate:"required"`
}

type UnlockAccountUserEndpoint struct{}

func (e *UnlockAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnlockAccountUserRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*UnlockAccountUserRequest, *apiresource.EmptyResource]{
		Title:             "Unlock Account User",
		Description:       "Unlocks a previously locked account user, restoring their access to the account.",
		Method:            http.MethodPost,
		Route:             "/v1/identity/account-users/{id}/unlock",
		Request:           &UnlockAccountUserRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnlockAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).UnlockAccountUser
		},
	}
}
