package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteAccountUserRequest is the request to delete an account user.
type DeleteAccountUserRequest struct {
	// The ID of the account user to delete.
	AccountUserID string `path:"id" validate:"required"`
}

type DeleteAccountUserEndpoint struct{}

func (e *DeleteAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountUserRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteAccountUserRequest, *apiresource.EmptyResource]{
		Title:             "Delete Account User",
		Description:       "Soft-deletes an account user, setting their status to removed.",
		Method:            http.MethodDelete,
		Route:             "/v1/identity/account-users/{id}",
		Request:           &DeleteAccountUserRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).DeleteAccountUser
		},
	}
}
