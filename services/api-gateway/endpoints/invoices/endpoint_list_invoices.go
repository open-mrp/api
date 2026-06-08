package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.Invoice], *apierror.APIError) {
			return svc.(InvoiceSvc).ListInvoices
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoice,
			Fields:     []string{"customer", "order", "shipment", "billing_address", "payment_term"},
		}),
	})
}
