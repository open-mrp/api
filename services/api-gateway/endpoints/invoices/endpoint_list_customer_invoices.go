package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListCustomerInvoicesRequest is the request to list invoices for a customer account.
type ListCustomerInvoicesRequest struct {
	apiresource.PaginationRequest
	// The customer account ID to list invoices for.
	CustomerAccountID string `path:"account_id" validate:"required"`
	// Whether to include child account invoices.
	IncludeChildAccounts bool `query:"include_child_accounts"`
}

type ListCustomerInvoicesEndpoint struct{}

func (e *ListCustomerInvoicesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomerInvoicesRequest, *apiresource.List[apiresource.InvoiceForPayment]] {
	return &apiendpoint.APIEndpoint[*ListCustomerInvoicesRequest, *apiresource.List[apiresource.InvoiceForPayment]]{
		Title:             "List Customer Invoices",
		Description:       "Returns a paginated list of invoices for a specific customer account, optionally including child account invoices.",
		Method:            http.MethodGet,
		Route:             "/v1/finance/accounts/{account_id}/invoices",
		Request:           &ListCustomerInvoicesRequest{},
		Response:          &apiresource.List[apiresource.InvoiceForPayment]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError) {
			return svc.(InvoiceSvc).ListCustomerInvoices
		},
	}
}
