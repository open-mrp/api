package invoiceep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list the open invoices of a customer account for payment.
type ListCustomerInvoicesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// ID of the customer account whose invoices are listed.
	CustomerAccountID string `path:"account_id" validate:"required"`
}

// Returns a paginated list of a customer's open invoices, newest first, in the shape used to apply a payment.
//
// Only invoices not marked paid in full are returned; an overpaid invoice is marked paid in full, so it is omitted too. Invoices billed to the customer's child accounts are included alongside its own, because the parent settles for them. Each invoice carries the payments already allocated to it, so the remaining balance can be worked out client-side.
//
// A free-text search term (`q`) is matched against the invoice number, the sales order number, and the customer PO number.
type ListCustomerInvoicesEndpoint struct{}

func (e *ListCustomerInvoicesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCustomerInvoicesRequest, *apiresource.List[apiresource.InvoiceForPayment]] {
	return (&apiendpoint.APIEndpoint[*ListCustomerInvoicesRequest, *apiresource.List[apiresource.InvoiceForPayment]]{
		Title:             "List Customer Invoices",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/accounts/{account_id}/invoices",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeInvoiceForPayment,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError) {
			return svc.(InvoiceSvc).ListCustomerInvoices
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoiceForPayment,
			Fields:     []string{"customer", "parent_account", "billing_address", "allocations", "allocations.amount", "allocations.amount.unit", "allocations.transaction", "allocations.transaction.amount", "allocations.transaction.amount.unit"},
		}),
	})
}
