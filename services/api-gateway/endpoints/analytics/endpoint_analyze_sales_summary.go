package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// AnalyzeSalesSummaryRequest is the request to total invoiced sales over a period.
type AnalyzeSalesSummaryRequest struct {
	// Start of the period, by invoice date, inclusive.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// End of the period, by invoice date, inclusive.
	EndDate time.Time `json:"ends_at" validate:"required"`
	SalesComparisonPeriod
	SalesReportFilters
	// The caller's UTC offset in minutes east of UTC (e.g. `-300` for US Eastern standard time). Daily totals are bucketed by the caller's local day. Defaults to `0`.
	TZOffsetMinutes field.Optional[int32] `json:"tz_offset_minutes,omitzero" validate:"omitempty,min=-840,max=840"`
}

// Returns what was invoiced over a period as one figure and day by day, and the same for a comparison period when one is given.
//
// Figures are computed from pre-priced invoice lines, so any window reads in about the same time. Cost is null for sales reps.
type AnalyzeSalesSummaryEndpoint struct{}

func (e *AnalyzeSalesSummaryEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeSalesSummaryRequest, *apiresource.AnalyzeSalesSummaryResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeSalesSummaryRequest, *apiresource.AnalyzeSalesSummaryResponse]{
		Title:               "Analyze Sales Summary",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/sales-summary",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeSalesSummaryRequest) (*apiresource.AnalyzeSalesSummaryResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeSalesSummary
		},
	})
}
