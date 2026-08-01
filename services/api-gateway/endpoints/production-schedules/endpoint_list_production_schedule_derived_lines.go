package productionscheduleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
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

// Returns the downstream department work implied by a schedule's constraint plan.
//
// The solver schedules only the constraint; every other department's work is derived from it by walking the production-step graph, applying each step's lead-time offset and yield. That makes this the work list a supervisor reads, rather than a second plan someone has to maintain.
//
// `explosion_depth` is how many steps downstream the work sits, which is what a readiness indicator keys off. Work whose derived week falls past the schedule's horizon is still returned — a department needs to see it coming.
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
