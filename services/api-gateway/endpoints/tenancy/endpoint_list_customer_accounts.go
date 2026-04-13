package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list customer accounts accessible to the authenticated user under a vendor account.
type ListCustomerAccountsRequest struct {
	// Vendor account ID.
	VendorAccountID string `path:"vendor_account_id" validate:"required"`
	apiresource.PaginationRequest
}

type ListCustomerAccountsEndpoint struct{}

func (e *ListCustomerAccountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomerAccountsRequest, *apiresource.List[apiresource.CustomerAccountSummary]] {
	return &apiendpoint.APIEndpoint[*ListCustomerAccountsRequest, *apiresource.List[apiresource.CustomerAccountSummary]]{
		Title:             "List Customer Accounts",
		Description:       "Returns a paginated list of customer accounts accessible to the authenticated user under the specified vendor account.",
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
