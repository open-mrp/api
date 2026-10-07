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
	// - `paid`: only invoices marked paid in full, overpaid ones included.
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
	// How `q` is matched. Defaults to `prefix`.
	//
	// - `prefix`: `q` matches the start of the invoice number, the sales order number, the customer PO number, and the customer's name, number, and alias. Notes are not searched.
	// - `contains`: `q` matches anywhere in those fields, and also in the invoice note and the customer's notes. Slower on accounts with many invoices.
	QMatch *constants.InvoiceSearchMatch `query:"q_match"`
	// Restricts results to invoices with any of these numbers, matched exactly.
	//
	// Up to 100 numbers. Unlike `q`, which matches the start of a number (or any part of it with `q_match=contains`) and also searches customers, order numbers and purchase order numbers, this finds exactly the invoices named.
	Numbers []string `query:"numbers" validate:"omitempty,max=100,dive,required,max=255"`
	// Only include invoices created on or after this date (`YYYY-MM-DD`, UTC). A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	StartDate *string `query:"starts_at" validate:"omitempty,date_filter"`
	// Only include invoices created on or before this date (`YYYY-MM-DD`, UTC), covering that whole day. A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	EndDate *string `query:"ends_at" validate:"omitempty,date_filter"`
}

// Returns a paginated list of invoices for the current account, newest first.
//
// A free-text search term (`q`) matches the start of the invoice number, the sales order number, the customer PO number, and the customer's name, number, and alias, and still respects the other filters. With `q_match=contains` it matches anywhere in those fields and also searches the invoice note and the customer's notes.
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
				"lines.item",
				"lines.quantity",
				"lines.quantity.unit",
				"lines.unit_price",
				"lines.unit_price.numerator_unit",
				"lines.unit_price.denominator_unit",
			},
		}),
	})
}
