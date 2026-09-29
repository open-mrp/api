package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// AnalyzeSalesInvoicesRequest is the request to list the invoices in a period with their invoiced sales.
type AnalyzeSalesInvoicesRequest struct {
	// Start of the period, by invoice date, inclusive.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// End of the period, by invoice date, inclusive.
	EndDate time.Time `json:"ends_at" validate:"required"`
	SalesReportFilters
	// Opaque cursor from a previous page's `next_page_url` or `previous_page_url`. Omit for the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of invoices to return.
	Limit int32 `query:"limit" default:"10" validate:"min=1,max=100"`
}

// Returns the invoices raised in a period, newest first, each with the revenue and distinct item count of its lines that match the filters. Invoices with no matching sale line are left out.
type AnalyzeSalesInvoicesEndpoint struct{}

func (e *AnalyzeSalesInvoicesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeSalesInvoicesRequest, *apiresource.List[apiresource.SalesInvoice]] {
	return (&apiendpoint.APIEndpoint[*AnalyzeSalesInvoicesRequest, *apiresource.List[apiresource.SalesInvoice]]{
		Title:               "Analyze Sales Invoices",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/sales-invoices",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeSalesInvoicesRequest) (*apiresource.List[apiresource.SalesInvoice], *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeSalesInvoices
		},
	})
}
