package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to restore a removed account user.
type RestoreAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
}

type RestoreAccountUserEndpoint struct{}

func (e *RestoreAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*RestoreAccountUserRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RestoreAccountUserRequest, *apiresource.EmptyResource]{
		Title:             "Restore Account User",
		Description:       "Restores a removed account user, setting their status to active.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}/restore",
		Request:           &RestoreAccountUserRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RestoreAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).RestoreAccountUser
		},
	}
}
