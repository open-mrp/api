package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListCustomerAccountsRequest is the request to list customer accounts for the authenticated user under a vendor account.
type ListCustomerAccountsRequest struct {
	// The vendor account ID to list customer accounts for.
	VendorAccountID string `path:"vendor_account_id" validate:"required"`
	apiresource.PaginationRequest
}

type ListCustomerAccountsEndpoint struct{}

func (e *ListCustomerAccountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomerAccountsRequest, *apiresource.List[apiresource.CustomerAccountSummary]] {
	return &apiendpoint.APIEndpoint[*ListCustomerAccountsRequest, *apiresource.List[apiresource.CustomerAccountSummary]]{
		Title:             "List Customer Accounts",
		Description:       "Returns the customer accounts accessible to the authenticated user under the specified vendor account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/me/tenancy/customer-accounts/{vendor_account_id}",
		Request:           &ListCustomerAccountsRequest{},
		Response:          &apiresource.List[apiresource.CustomerAccountSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomerAccountsRequest) (*apiresource.List[apiresource.CustomerAccountSummary], *apierror.APIError) {
			return svc.(TenancySvc).ListCustomerAccounts
		},
	}
}
