package productionscheduleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list a schedule's derived department work.
type ListProductionScheduleDerivedLinesRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Only return work owned by these departments.
	DepartmentIDs []string `query:"department_ids"`
	// Only return work in this horizon week, zero-based.
	WeekIndex *int32 `query:"week_index"`
}

// Returns the department work implied by a schedule's constraint plan.
//
// The solver schedules only the constraint; every other department's work is derived from it by walking the production-step graph, applying each step's lead-time offset and yield. That makes this the work list a supervisor reads, rather than a second plan someone has to maintain.
//
// `explosion_depth` is how many steps downstream the work sits, which is what a readiness indicator keys off. Depth 0 is the constraint's own campaigns, so a plant with nothing configured downstream of its constraint still gets the work it actually scheduled. Work whose derived week falls past the schedule's horizon is still returned — a department needs to see it coming.
type ListProductionScheduleDerivedLinesEndpoint struct{}

func (e *ListProductionScheduleDerivedLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionScheduleDerivedLinesRequest, *apiresource.List[apiresource.ProductionScheduleDerivedLine]] {
	return (&apiendpoint.APIEndpoint[*ListProductionScheduleDerivedLinesRequest, *apiresource.List[apiresource.ProductionScheduleDerivedLine]]{
		Title:             "List Production Schedule Derived Lines",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/derived-lines",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleDerivedLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionScheduleDerivedLinesRequest) (*apiresource.List[apiresource.ProductionScheduleDerivedLine], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListProductionScheduleDerivedLines
		},
	})
}
