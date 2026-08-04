package childaccountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list child accounts.
type ListChildAccountsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of the accounts linked directly beneath the target account.
//
// Only direct children are returned, not children of those children. Results are ordered by when your customer record for each child was created, newest first, and the `q` search term matches the child account's name.
type ListChildAccountsEndpoint struct{}

func (e *ListChildAccountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListChildAccountsRequest, *apiresource.List[apiresource.ChildAccount]] {
	return (&apiendpoint.APIEndpoint[*ListChildAccountsRequest, *apiresource.List[apiresource.ChildAccount]]{
		Title:             "List Child Accounts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/child-accounts",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeChildAccount,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListChildAccountsRequest) (*apiresource.List[apiresource.ChildAccount], *apierror.APIError) {
			return svc.(ChildAccountSvc).ListChildAccounts
		},
	})
}
