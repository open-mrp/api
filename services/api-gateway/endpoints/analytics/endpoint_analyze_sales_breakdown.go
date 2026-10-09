package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// AnalyzeSalesBreakdownRequest is the request to total invoiced sales by one dimension.
type AnalyzeSalesBreakdownRequest struct {
	// The dimension to total by.
	GroupBy constants.SalesBreakdownGroupBy `json:"group_by" validate:"required"`
	// Start of the period, by invoice date, inclusive.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// End of the period, by invoice date, inclusive.
	EndDate time.Time `json:"ends_at" validate:"required"`
	SalesComparisonPeriod
	SalesReportFilters
	// Opaque cursor from a previous page's `next_page_url` or `previous_page_url`. Omit for the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of groups to return.
	Limit int32 `query:"limit" default:"10" validate:"min=1,max=100"`
}

// Returns invoiced sales totalled by customer, product, product line, customer group, sales rep or discount, largest current-period revenue first.
//
// Each group carries its current-period totals and, when a comparison period is given, the same group's comparison totals, so the two line up row for row. A group with no current-period revenue is left out, except when grouping by product, where a product invoiced at zero is kept. Customer groups, sales reps and discounts leave out sales that have none.
type AnalyzeSalesBreakdownEndpoint struct{}

func (e *AnalyzeSalesBreakdownEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeSalesBreakdownRequest, *apiresource.List[apiresource.SalesBreakdown]] {
	return (&apiendpoint.APIEndpoint[*AnalyzeSalesBreakdownRequest, *apiresource.List[apiresource.SalesBreakdown]]{
		Title:               "Analyze Sales Breakdown",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/sales-breakdown",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeSalesBreakdownRequest) (*apiresource.List[apiresource.SalesBreakdown], *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeSalesBreakdown
		},
	})
}
