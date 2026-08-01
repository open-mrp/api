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

// Request to list production schedules.
type ListProductionSchedulesRequest struct {
	apiresource.PaginationRequest
	// Only return versions in these lifecycle states.
	Statuses []constants.ProductionScheduleStatus `query:"statuses"`
}

// Returns a paginated list of production schedule versions, newest first.
type ListProductionSchedulesEndpoint struct{}

func (e *ListProductionSchedulesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionSchedulesRequest, *apiresource.List[apiresource.ProductionSchedule]] {
	return (&apiendpoint.APIEndpoint[*ListProductionSchedulesRequest, *apiresource.List[apiresource.ProductionSchedule]]{
		Title:             "List Production Schedules",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionSchedulesRequest) (*apiresource.List[apiresource.ProductionSchedule], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListProductionSchedules
		},
	})
}
