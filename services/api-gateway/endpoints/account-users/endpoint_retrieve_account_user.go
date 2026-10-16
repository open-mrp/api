package accountuserep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve an account user.
type RetrieveAccountUserRequest struct {
	// ID of the account user to retrieve.
	AccountUserID string `path:"id" validate:"required"`
}

// Retrieves an account user, whatever their status.
//
// Only members of the account you are acting in are found; another account's member is a 404.
type RetrieveAccountUserEndpoint struct{}

func (e *RetrieveAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountUserRequest, *apiresource.AccountUser] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Retrieve Account User",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).GetAccountUser
		},
		ObjectType:    constants.ObjectTypeAccountUser,
		IncludeConfig: accountUserIncludes(),
	})
}
