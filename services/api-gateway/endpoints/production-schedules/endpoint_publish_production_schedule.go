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

// Request to publish a production schedule.
type PublishProductionScheduleRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
}

// Publishes a draft schedule, freezing its first weeks.
//
// Publishing is what makes a plan a commitment: the frozen weeks' lines are marked frozen, the frozen line count and quantity are captured onto the version, and any published version covering the same horizon is superseded rather than rewritten. After this, changes inside the frozen window require a reason and are recorded as deviations.
//
// The frozen counts are snapshotted here and never recomputed, so adherence keeps the denominator it was committed to.
type PublishProductionScheduleEndpoint struct{}

func (e *PublishProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*PublishProductionScheduleRequest, *apiresource.ProductionSchedule] {
	return (&apiendpoint.APIEndpoint[*PublishProductionScheduleRequest, *apiresource.ProductionSchedule]{
		Title:             "Publish Production Schedule",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/actions/publish",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PublishProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).PublishProductionSchedule
		},
	})
}
