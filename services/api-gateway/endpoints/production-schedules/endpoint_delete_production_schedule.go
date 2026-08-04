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

// Request to delete a production schedule.
type DeleteProductionScheduleRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
}

// Deletes a draft schedule along with its planned campaigns and its item policy snapshot.
//
// Only drafts can be deleted. A published version is the baseline attainment is measured against, so removing it would erase the record of what was promised — archive those instead.
type DeleteProductionScheduleEndpoint struct{}

func (e *DeleteProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionScheduleRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteProductionScheduleRequest, *apiresource.EmptyResource]{
		Title:             "Delete Production Schedule",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionScheduleRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).DeleteProductionSchedule
		},
	})
}
