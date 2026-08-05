package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeOeeRequest is the request to analyze Overall Equipment Effectiveness (OEE).
type AnalyzeOeeRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// Optional department IDs to filter by.
	DepartmentIDs []string `json:"department_ids,omitzero"`
	// Scheduled production time per department for the period. Availability, performance and OEE are only returned for departments this covers.
	PlannedTime []OeeDepartmentPlannedTime `json:"planned_time,omitzero"`
}

// OeeDepartmentPlannedTime supplies the scheduled production time for one department.
type OeeDepartmentPlannedTime struct {
	// The department ID.
	DepartmentID string `json:"department_id" validate:"required"`
	// Scheduled production hours for the period.
	PlannedHours float64 `json:"planned_hours" validate:"required"`
}

// Returns Overall Equipment Effectiveness (OEE) metrics by department.
//
// Availability is measured from logged machine downtime rather than inferred, so it requires both `planned_time` for the department and downtime events in the period. Departments with `has_downtime_data` false have no availability measurement, and their ratios are returned as null rather than as 100%.
type AnalyzeOeeEndpoint struct{}

func (e *AnalyzeOeeEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOeeRequest, *apiresource.AnalyzeOeeResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeOeeRequest, *apiresource.AnalyzeOeeResponse]{
		Title:               "Analyze OEE",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/oee",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOeeRequest) (*apiresource.AnalyzeOeeResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOee
		},
	})
}
