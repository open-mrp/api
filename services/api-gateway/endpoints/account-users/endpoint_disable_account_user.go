package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to disable an account user.
type DisableAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
}

// Disables an account user. Disabled users will not be able to access the target account.
type DisableAccountUserEndpoint struct{}

func (e *DisableAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*DisableAccountUserRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DisableAccountUserRequest, *apiresource.EmptyResource]{
		Title:             "Disable Account User",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}/actions/disable",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DisableAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).DisableAccountUser
		},
	})
}
