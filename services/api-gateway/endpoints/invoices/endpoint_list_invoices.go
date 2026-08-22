package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list invoices.
type ListInvoicesRequest struct {
	apiresource.PaginationRequest
	// Restricts results to invoices in this payment state.
	//
	// - `all`: no payment-state filtering, the same as omitting the parameter.
	// - `paid`: only invoices marked paid in full.
	// - `unpaid`: only invoices not marked paid in full, including invoices carrying partial payments.
	// - `overpaid`: only invoices whose applied payments exceed the invoiced amount.
	Status *constants.InvoiceListStatus `query:"status"`
	// Restricts results to invoices whose sales order has at least one line for any of these items.
	ItemIDs []string `query:"item_ids"`
	// Restricts results to invoices billed to any of these customers.
	CustomerIDs []string `query:"customer_ids"`
	// Restricts results to invoices whose sales order has at least one line whose product belongs to any of these product lines.
	ProductLineIDs []string `query:"product_line_ids"`
	// Restricts results to invoices billed to customers belonging to any of these account groups.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Restricts results to invoices whose sales order is credited to any of these sales reps.
	//
	// These are account user IDs, matching the `sales_rep` on the order.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Earliest invoice creation date to include, in `YYYY-MM-DD` format.
	StartDate *string `query:"starts_at"`
	// Latest invoice creation date to include, in `YYYY-MM-DD` format.
	//
	// Compared against the creation timestamp at the start of that day, so invoices created later on the end date itself are excluded; pass the following day to include them.
	EndDate *string `query:"ends_at"`
}

// Returns a paginated list of invoices for the current account, newest first.
//
// A free-text search term (`q`) is matched against the invoice number, the invoice note, the customer name, the sales order number, the customer PO number, and the customer number, and still respects the other filters.
type ListInvoicesEndpoint struct{}

func (e *ListInvoicesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInvoicesRequest, *apiresource.List[apiresource.Invoice]] {
	return (&apiendpoint.APIEndpoint[*ListInvoicesRequest, *apiresource.List[apiresource.Invoice]]{
		Title:             "List Invoices",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/invoices",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeInvoice,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.Invoice], *apierror.APIError) {
			return svc.(InvoiceSvc).ListInvoices
		},
		// Same resource as retrieve, so the same include set — allocations are the exception: only
		// the retrieve and update RPCs expand them.
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoice,
			Fields: []string{
				"customer",
				"order",
				"shipment",
				"related.sales_order",
				"related.shipment",
				"billing_address",
				"payment_term",
				"lines",
				"lines.order_line",
				"lines.order_line.product",
			},
		}),
	})
}
