package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list account users.
type ListAccountUsersRequest struct {
	apiresource.PaginationRequest
	// Filter by role type.
	//
	// - `admin`: account administrators.
	// - `user`: users with a custom role.
	// - `scanner`: scanning station users.
	// - `sales_rep`: sales representatives.
	// - `agent`: automated agents.
	RoleType *constants.RoleType `query:"role_type"`
	// Controls whether removed (soft-deleted) account users appear in the list.
	//
	// - `excluded`: only active and disabled users (default).
	// - `included`: removed users are listed as well.
	RemovedScope *constants.RemovedResourceScope `query:"removed_scope"`
}

// Returns a paginated list of account users for the current account.
type ListAccountUsersEndpoint struct{}

func (e *ListAccountUsersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountUsersRequest, *apiresource.List[apiresource.AccountUser]] {
	return (&apiendpoint.APIEndpoint[*ListAccountUsersRequest, *apiresource.List[apiresource.AccountUser]]{
		Title:               "List Account Users",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountUsersRequest) (*apiresource.List[apiresource.AccountUser], *apierror.APIError) {
			return svc.(AccountUserSvc).ListAccountUsers
		},
		ObjectType: constants.ObjectTypeAccountUser,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"user", "role", "department"},
		}),
	})
}
