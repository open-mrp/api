package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an account user.
type RetrieveAccountUserRequest struct {
	// Account user ID.
	AccountUserID string `path:"id" validate:"required"`
}

type RetrieveAccountUserEndpoint struct{}

func (e *RetrieveAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountUserRequest, *apiresource.AccountUser] {
	return &apiendpoint.APIEndpoint[*RetrieveAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Retrieve Account User",
		Description:       "Returns an account user by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}",
		Request:           &RetrieveAccountUserRequest{},
		Response:          &apiresource.AccountUser{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).GetAccountUser
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"role", "department"},
		}),
	}
}
