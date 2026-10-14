package productionscheduleep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductionScheduleRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteProductionScheduleRequest, *object.Deleted]{
		Title:             "Delete Production Schedule",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductionScheduleRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ProductionScheduleSvc).DeleteProductionSchedule, constants.ObjectTypeProductionSchedule, "id")
		},
	})
}
