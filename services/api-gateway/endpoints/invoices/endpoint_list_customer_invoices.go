package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list invoices for a customer account.
type ListCustomerInvoicesRequest struct {
	apiresource.PaginationRequest
	// Customer account ID.
	CustomerAccountID string `path:"account_id" validate:"required"`
	// Whether to also include invoices billed to the customer's child accounts.
	//
	// Currently has no effect: invoices for child accounts are always included.
	IncludeChildAccounts bool `query:"include_child_accounts"`
}

// Returns a paginated list of payment-oriented invoices for a specific customer account, including invoices billed to its child accounts.
type ListCustomerInvoicesEndpoint struct{}

func (e *ListCustomerInvoicesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomerInvoicesRequest, *apiresource.List[apiresource.InvoiceForPayment]] {
	return (&apiendpoint.APIEndpoint[*ListCustomerInvoicesRequest, *apiresource.List[apiresource.InvoiceForPayment]]{
		Title:             "List Customer Invoices",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/accounts/{account_id}/invoices",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeInvoiceForPayment,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError) {
			return svc.(InvoiceSvc).ListCustomerInvoices
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoiceForPayment,
			Fields:     []string{"customer", "parent_account"},
		}),
	})
}
