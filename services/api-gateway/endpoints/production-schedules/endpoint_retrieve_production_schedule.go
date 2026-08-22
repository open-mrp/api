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

// Request to retrieve a production schedule.
type RetrieveProductionScheduleRequest struct {
	// ID of the schedule version.
	ProductionScheduleID string `path:"id" validate:"required"`
}

// Returns a single production schedule version.
type RetrieveProductionScheduleEndpoint struct{}

func (e *RetrieveProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductionScheduleRequest, *apiresource.ProductionSchedule] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductionScheduleRequest, *apiresource.ProductionSchedule]{
		Title:             "Retrieve Production Schedule",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).GetProductionSchedule
		},
	})
}
