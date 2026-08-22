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
	"github.com/open-mrp/api/shared/field"
)

// AnalyzeScheduleAttainmentRequest is the request to measure production against plan.
type AnalyzeScheduleAttainmentRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// The dimension to break the results down by. Defaults to `week`.
	GroupBy field.Optional[constants.AttainmentGroupBy] `json:"group_by,omitzero"`
	// Only measure production on these machines.
	MachineIDs []string `json:"machine_ids,omitzero"`
	// Only measure production in these departments.
	DepartmentIDs []string `json:"department_ids,omitzero"`
}

// Returns actual production measured against the plan that was live at the time.
//
// The baseline for each week is the schedule version published on or before that week began, so republishing mid-horizon cannot rewrite a week the floor has already worked. `baseline_schedules` names the versions used.
//
// Two ratios are returned because either alone misleads: `attainment_pct` caps each campaign at what was asked for, so over-building one SKU cannot hide a miss on another, while `output_ratio_pct` is uncapped and is what reveals over-production. Production with no matching planned campaign is reported as `unplanned_quantity` rather than discarded — that number is the clearest signal a schedule is being worked around.
//
// Every ratio is null rather than zero when nothing was planned, and `has_baseline` is false when nothing was ever published over the period.
type AnalyzeScheduleAttainmentEndpoint struct{}

func (e *AnalyzeScheduleAttainmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeScheduleAttainmentRequest, *apiresource.AnalyzeScheduleAttainmentResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeScheduleAttainmentRequest, *apiresource.AnalyzeScheduleAttainmentResponse]{
		Title:               "Analyze Schedule Attainment",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/schedule-attainment",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		AgentTool:           true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeScheduleAttainmentRequest) (*apiresource.AnalyzeScheduleAttainmentResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeScheduleAttainment
		},
	})
}
