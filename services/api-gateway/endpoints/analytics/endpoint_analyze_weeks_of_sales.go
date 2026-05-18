package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeWeeksOfSalesRequest is the request to analyze weeks of sales.
type AnalyzeWeeksOfSalesRequest struct {
	// The number of weeks to use for the sales period. Defaults to 4, minimum 1.
	PeriodInWeeks *int32 `query:"period_in_weeks"`
}

// Returns weeks-of-sales metrics per product line, including on-hand quantity, average weekly sales, and weeks of inventory remaining.
type AnalyzeWeeksOfSalesEndpoint struct{}

func (e *AnalyzeWeeksOfSalesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeWeeksOfSalesRequest, *apiresource.AnalyzeWeeksOfSalesResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeWeeksOfSalesRequest, *apiresource.AnalyzeWeeksOfSalesResponse]{
		Title:             "Analyze Weeks of Sales",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/analytics/weeks-of-sales",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeWeeksOfSalesRequest) (*apiresource.AnalyzeWeeksOfSalesResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeWeeksOfSales
		},
	})
}
