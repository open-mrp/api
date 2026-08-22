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

// AnalyzeOeeTrendRequest is the request to analyze Overall Equipment Effectiveness (OEE) over time.
type AnalyzeOeeTrendRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// Restrict the analysis to these departments.
	DepartmentIDs []string `json:"department_ids,omitzero"`
}

// Returns Overall Equipment Effectiveness (OEE) by production week.
//
// Each period carries the same four terms `/v1/core/analytics/oee` reports for a single window, rolled up across departments and weighted by seconds rather than averaged, so a department that ran for an hour does not weigh as heavily as one that ran all week. Weeks start on Monday, and the first and last period of a window are clipped to the window itself.
//
// Only departments with scheduled time take part: a department with no machines has no availability, so counting its output in quality would leave the three terms describing different plants. Compare two windows by calling this twice.
type AnalyzeOeeTrendEndpoint struct{}

func (e *AnalyzeOeeTrendEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOeeTrendRequest, *apiresource.AnalyzeOeeTrendResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeOeeTrendRequest, *apiresource.AnalyzeOeeTrendResponse]{
		Title:               "Analyze OEE trend",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/oee-trend",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		ReadOnly:            true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOeeTrendRequest) (*apiresource.AnalyzeOeeTrendResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOeeTrend
		},
	})
}
