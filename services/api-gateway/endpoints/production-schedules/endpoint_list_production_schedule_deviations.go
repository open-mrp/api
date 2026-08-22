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

// Request to list a schedule's deviations.
type ListProductionScheduleDeviationsRequest struct {
	apiresource.PaginationRequest
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Whether the change fell inside the frozen window.
	//
	// Judged against the freeze as it stood when the change was made, not as it stands now, so a later publish cannot reclassify history.
	Frozen *bool `query:"frozen"`
}

// Returns the append-only log of hand changes made to a schedule, most recent first.
//
// This is what frozen-week adherence is measured from. A change recorded as frozen was inside the freeze window at the moment it was made, and stays that way regardless of what is published later.
type ListProductionScheduleDeviationsEndpoint struct{}

func (e *ListProductionScheduleDeviationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionScheduleDeviationsRequest, *apiresource.List[apiresource.ProductionScheduleDeviation]] {
	return (&apiendpoint.APIEndpoint[*ListProductionScheduleDeviationsRequest, *apiresource.List[apiresource.ProductionScheduleDeviation]]{
		Title:             "List Production Schedule Deviations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/deviations",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleDeviation,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionScheduleDeviationsRequest) (*apiresource.List[apiresource.ProductionScheduleDeviation], *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ListProductionScheduleDeviations
		},
	})
}
