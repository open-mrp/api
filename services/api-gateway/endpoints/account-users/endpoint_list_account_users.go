package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListAccountUsersRequest is the request to list account users with optional filters.
type ListAccountUsersRequest struct {
	apiresource.PaginationRequest
	// Filter by role type code.
	RoleType *constants.RoleTypeCode `query:"role_type"`
	// Whether to include removed account users in the results.
	IncludeRemoved bool `query:"include_removed"`
}

type ListAccountUsersEndpoint struct{}

func (e *ListAccountUsersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountUsersRequest, *apiresource.List[apiresource.AccountUser]] {
	return &apiendpoint.APIEndpoint[*ListAccountUsersRequest, *apiresource.List[apiresource.AccountUser]]{
		Title:             "List Account Users",
		Description:       "Returns a paginated list of account users for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/account-users",
		Request:           &ListAccountUsersRequest{},
		Response:          &apiresource.List[apiresource.AccountUser]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountUsersRequest) (*apiresource.List[apiresource.AccountUser], *apierror.APIError) {
			return svc.(AccountUserSvc).ListAccountUsers
		},
	}
}
