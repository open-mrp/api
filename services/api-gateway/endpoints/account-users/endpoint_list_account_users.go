package accountuserep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list account users.
type ListAccountUsersRequest struct {
	object.ListRequest
	apirequest.CreatedRange
	// Case-insensitive substring match on the person's `name`, `email` or `username`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return members with any of these statuses: `active`, `disabled` or `removed`. Leave it out to list every status.
	Statuses []constants.AccountUserStatus `query:"statuses"`
	// Only return members whose role has this type: `admin`, `user`, `scanner`, `sales_rep` or `agent`.
	RoleType *constants.RoleType `query:"role_type"`
	// Only return members who can (`eligible`) or can't (`ineligible`) be named as a sales rep.
	CommissionEligibility *constants.CommissionEligibility `query:"commission_eligibility"`
}

// Returns a list of the people in the account you are acting in.
//
// Sorted by creation date, newest first. Acting in a customer or supplier account you manage lists that account's contacts rather than your own team.
type ListAccountUsersEndpoint struct{}

func (e *ListAccountUsersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountUsersRequest, *apiresource.List[apiresource.AccountUser]] {
	return (&apiendpoint.APIEndpoint[*ListAccountUsersRequest, *apiresource.List[apiresource.AccountUser]]{
		Title:             "List Account Users",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountUsersRequest) (*apiresource.List[apiresource.AccountUser], *apierror.APIError) {
			return svc.(AccountUserSvc).ListAccountUsers
		},
		ObjectType:    constants.ObjectTypeAccountUser,
		IncludeConfig: accountUserIncludes(),
	})
}

// accountUserIncludeFields are the include keys every account-user endpoint offers.
var accountUserIncludeFields = []string{"user", "role", "work_center"}

func accountUserIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeAccountUser,
		Fields:     accountUserIncludeFields,
	})
}
