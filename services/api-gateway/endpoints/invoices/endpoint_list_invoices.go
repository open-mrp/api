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

// Request to list invoices.
type ListInvoicesRequest struct {
	apiresource.PaginationRequest
	// Filter invoices by payment status.
	//
	// - `all`: no payment-status filtering (same as omitting the parameter).
	// - `paid`: only invoices paid in full.
	// - `unpaid`: only invoices that are neither paid in full nor overpaid.
	// - `overpaid`: only invoices whose allocations exceed the invoiced amount.
	Status *string `query:"status" validate:"omitempty,oneof=all paid unpaid overpaid"`
	// Filter by item IDs present in invoice lines.
	ItemIDs []string `query:"item_ids"`
	// Filter by customer account IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by product line IDs.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by customer group IDs.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Filter by sales rep user IDs.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Only return invoices created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"start_date"`
	// Only return invoices created before this date (`YYYY-MM-DD`).
	EndDate *string `query:"end_date"`
}

// Returns a paginated list of invoices for the current account.
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
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoice,
			Fields:     []string{"customer", "order", "shipment", "billing_address", "payment_term", "lines"},
		}),
	})
}
