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

// Request to archive a production schedule.
type ArchiveProductionScheduleRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
}

// Archives a schedule version, retiring it without discarding its history.
//
// This is how a published version is taken out of use: it stays readable and still backs any attainment already measured against it.
type ArchiveProductionScheduleEndpoint struct{}

func (e *ArchiveProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*ArchiveProductionScheduleRequest, *apiresource.ProductionSchedule] {
	return (&apiendpoint.APIEndpoint[*ArchiveProductionScheduleRequest, *apiresource.ProductionSchedule]{
		Title:             "Archive Production Schedule",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/actions/archive",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ArchiveProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ArchiveProductionSchedule
		},
	})
}
