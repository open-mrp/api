package childaccountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list child accounts.
type ListChildAccountsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of child accounts for the target account.
type ListChildAccountsEndpoint struct{}

func (e *ListChildAccountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListChildAccountsRequest, *apiresource.List[apiresource.ChildAccount]] {
	return (&apiendpoint.APIEndpoint[*ListChildAccountsRequest, *apiresource.List[apiresource.ChildAccount]]{
		Title:             "List Child Accounts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/child-accounts",
		Request:           &ListChildAccountsRequest{},
		Response:          &apiresource.List[apiresource.ChildAccount]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListChildAccountsRequest) (*apiresource.List[apiresource.ChildAccount], *apierror.APIError) {
			return svc.(ChildAccountSvc).ListChildAccounts
		},
	}).WithDocSource(e)
}
