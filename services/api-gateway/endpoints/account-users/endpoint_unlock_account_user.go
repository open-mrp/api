package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to unlock an account user.
type UnlockAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
}

type UnlockAccountUserEndpoint struct{}

func (e *UnlockAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnlockAccountUserRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*UnlockAccountUserRequest, *apiresource.EmptyResource]{
		Title:             "Unlock Account User",
		Description:       "Unlocks an account user, restoring their access to the account.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
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
