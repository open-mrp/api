package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list account users.
type ListAccountUsersRequest struct {
	apiresource.PaginationRequest
	// Filter by role type code.
	RoleType *constants.RoleType `query:"role_type"`
	// Controls whether removed account users are included.
	RemovedScope *constants.RemovedResourceScope `query:"removed_scope"`
}

// Returns a paginated list of account users for the current account.
type ListAccountUsersEndpoint struct{}

func (e *ListAccountUsersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountUsersRequest, *apiresource.List[apiresource.AccountUser]] {
	return (&apiendpoint.APIEndpoint[*ListAccountUsersRequest, *apiresource.List[apiresource.AccountUser]]{
		Title:             "List Account Users",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users",
		Request:           &ListAccountUsersRequest{},
		Response:          &apiresource.List[apiresource.AccountUser]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountUsersRequest) (*apiresource.List[apiresource.AccountUser], *apierror.APIError) {
			return svc.(AccountUserSvc).ListAccountUsers
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"role", "department"},
		}),
	}).WithDocSource(e)
}
