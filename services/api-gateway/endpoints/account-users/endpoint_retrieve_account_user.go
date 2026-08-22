package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an account user.
type RetrieveAccountUserRequest struct {
	// ID of the account user to retrieve.
	AccountUserID string `path:"id" validate:"required"`
}

// Returns an account user by ID.
//
// The lookup is scoped to the account you are acting in, so an ID belonging to another account is reported as not found.
type RetrieveAccountUserEndpoint struct{}

func (e *RetrieveAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountUserRequest, *apiresource.AccountUser] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountUserRequest, *apiresource.AccountUser]{
		Title:               "Retrieve Account User",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).GetAccountUser
		},
		ObjectType: constants.ObjectTypeAccountUser,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"user", "role", "department"},
		}),
	})
}
