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

// Request to retrieve the current production schedule.
type RetrieveCurrentProductionScheduleRequest struct{}

// Returns the published schedule covering today.
//
// Responds 404 when no published version covers today, which is the normal state before the first schedule is published.
type RetrieveCurrentProductionScheduleEndpoint struct{}

func (e *RetrieveCurrentProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveCurrentProductionScheduleRequest, *apiresource.ProductionSchedule] {
	return (&apiendpoint.APIEndpoint[*RetrieveCurrentProductionScheduleRequest, *apiresource.ProductionSchedule]{
		Title:             "Retrieve Current Production Schedule",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/current",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCurrentProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).GetCurrentProductionSchedule
		},
	})
}
