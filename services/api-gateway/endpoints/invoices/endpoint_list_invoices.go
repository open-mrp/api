package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list invoices.
type ListInvoicesRequest struct {
	apiresource.PaginationRequest
	// Filter by status: all, paid, unpaid, or overpaid.
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
	// Filter by start date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter by end date (inclusive).
	EndDate *string `query:"end_date"`
}

// TODO: stop returning InvoiceSummary; return the full Invoice apiresource and use proper includes values to control expansion.

// Returns a paginated list of invoices for the current account.
type ListInvoicesEndpoint struct{}

func (e *ListInvoicesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInvoicesRequest, *apiresource.List[apiresource.InvoiceSummary]] {
	return (&apiendpoint.APIEndpoint[*ListInvoicesRequest, *apiresource.List[apiresource.InvoiceSummary]]{
		Title:             "List Invoices",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/invoices",
		Request:           &ListInvoicesRequest{},
		Response:          &apiresource.List[apiresource.InvoiceSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.InvoiceSummary], *apierror.APIError) {
			return svc.(InvoiceSvc).ListInvoices
		},
	}).WithDocSource(e)
}
