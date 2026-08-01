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

// Request to remove a campaign from a schedule.
type DeleteProductionScheduleLineRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// ID of the schedule line.
	LineID string `path:"line_id" validate:"required"`
	// Why the campaign was removed. Required when it sits in a frozen week.
	Reason *constants.ScheduleChangeReason `query:"reason"`
	// Free-form explanation of the change.
	ReasonNote *string `query:"reason_note"`
}

// Removes a campaign from a schedule.
//
// The deviation log keeps a full snapshot of the removed line, so the change stays readable after the line itself is gone. Removing from a frozen week requires a `reason`.
type DeleteProductionScheduleLineEndpoint struct{}

func (e *DeleteProductionScheduleLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionScheduleLineRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteProductionScheduleLineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Production Schedule Line",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionScheduleLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).DeleteProductionScheduleLine
		},
	})
}
