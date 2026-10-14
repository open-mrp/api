package tenancyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list customer accounts accessible to the authenticated user under a seller account.
type ListCustomerAccountsRequest struct {
	// ID of the seller account whose customers to list.
	VendorAccountID string `path:"vendor_account_id" validate:"required"`
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// TODO: stop returning CustomerAccountSummary; return the full CustomerAccount apiresource and use proper includes values to control expansion.

// Returns the customer accounts of the given seller account that the authenticated user belongs to.
//
// This is how a buyer with access to more than one of a seller's customer accounts chooses which one to act as. Only accounts where the user's membership is still active are returned. The paging and search parameters are ignored: every match comes back in a single page.
type ListCustomerAccountsEndpoint struct{}

func (e *ListCustomerAccountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomerAccountsRequest, *apiresource.List[apiresource.CustomerAccountSummary]] {
	return (&apiendpoint.APIEndpoint[*ListCustomerAccountsRequest, *apiresource.List[apiresource.CustomerAccountSummary]]{
		Title:             "List Customer Accounts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/me/tenancy/customer-accounts/{vendor_account_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomerAccountsRequest) (*apiresource.List[apiresource.CustomerAccountSummary], *apierror.APIError) {
			return svc.(TenancySvc).ListCustomerAccounts
		},
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
	})
}
