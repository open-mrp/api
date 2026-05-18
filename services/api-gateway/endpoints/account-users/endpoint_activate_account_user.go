package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to activate an account user.
type ActivateAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
}

// Activates a disabled or removed account user.
type ActivateAccountUserEndpoint struct{}

func (e *ActivateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*ActivateAccountUserRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ActivateAccountUserRequest, *apiresource.EmptyResource]{
		Title:             "Activate Account User",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}/actions/activate",
		Request:           &ActivateAccountUserRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ActivateAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).ActivateAccountUser
		},
	}).WithDocSource(e)
}
