package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to lock an account user.
type LockAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
}

type LockAccountUserEndpoint struct{}

func (e *LockAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*LockAccountUserRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*LockAccountUserRequest, *apiresource.EmptyResource]{
		Title:             "Lock Account User",
		Description:       "Locks an account user, preventing them from accessing the account.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}/lock",
		Request:           &LockAccountUserRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *LockAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).LockAccountUser
		},
	}
}
