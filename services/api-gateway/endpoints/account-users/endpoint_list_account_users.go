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
	// Filter by commission eligibility.
	//
	// Exact match on the column. Pass `true` to list users who can be assigned as sales representatives, including dedicated `sales_rep` users.
	IsCommissionEligible *bool `query:"is_commission_eligible"`
	// Controls whether removed (soft-deleted) account users appear in the list.
	//
	// Removed users are left out unless you pass `included`, so a user removed with the remove action disappears from the default listing.
	RemovedScope *constants.RemovedResourceScope `query:"removed_scope"`
}

// Returns a paginated list of the users who belong to the account you are acting in.
//
// When the account you are acting in is a customer or supplier account you manage, this lists that account's users rather than your own team.
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
