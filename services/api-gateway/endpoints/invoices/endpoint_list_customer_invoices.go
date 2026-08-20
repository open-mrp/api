package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the open invoices of a customer account for payment.
type ListCustomerInvoicesRequest struct {
	apiresource.PaginationRequest
	// ID of the customer account whose invoices are listed.
	CustomerAccountID string `path:"account_id" validate:"required"`
}

// Returns a paginated list of a customer's open invoices, newest first, in the shape used to apply a payment.
//
// Only invoices that still owe a balance are returned; invoices marked paid in full are omitted, while overpaid ones are kept because they still need correcting. Invoices billed to the customer's child accounts are included alongside its own, because the parent settles for them. Each invoice carries the payments already allocated to it, so the remaining balance can be worked out client-side.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError) {
			return svc.(InvoiceSvc).ListCustomerInvoices
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoiceForPayment,
			Fields:     []string{"customer", "parent_account", "allocations"},
		}),
	})
}
