package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetAccountUserRequest is the request to retrieve a single account user.
type GetAccountUserRequest struct {
	// The ID of the account user to retrieve.
	AccountUserID string `path:"id" validate:"required"`
}

type GetAccountUserEndpoint struct{}

func (e *GetAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountUserRequest, *apiresource.AccountUser] {
	return &apiendpoint.APIEndpoint[*GetAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Get Account User",
		Description:       "Returns a single account user by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/account-users/{id}",
		Request:           &GetAccountUserRequest{},
		Response:          &apiresource.AccountUser{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).GetAccountUser
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"role", "department"},
		}),
	}
}
