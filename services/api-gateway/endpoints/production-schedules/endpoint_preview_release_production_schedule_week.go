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

// Request to preview releasing one week of a production schedule.
type PreviewReleaseProductionScheduleWeekRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Zero-based week offset from the start of the horizon.
	WeekIndex int32 `query:"week_index" validate:"min=0"`
}

// Returns what releasing a week would create, without creating it.
//
// The lots are resolved exactly as the release itself resolves them, so what a planner is shown and what the floor receives cannot drift apart.
//
// `is_releasable` is false when the week is empty or already released, with `blocked_reason` saying which; `existing_production_run_id` names the run a released week is already tied to.
//
// Cancelled campaigns and campaigns planned at zero are excluded here exactly as the release excludes them, so a week holding nothing but those previews as empty.
type PreviewReleaseProductionScheduleWeekEndpoint struct{}

func (e *PreviewReleaseProductionScheduleWeekEndpoint) Materialize() *apiendpoint.APIEndpoint[*PreviewReleaseProductionScheduleWeekRequest, *apiresource.ReleaseScheduleWeekPreview] {
	return (&apiendpoint.APIEndpoint[*PreviewReleaseProductionScheduleWeekRequest, *apiresource.ReleaseScheduleWeekPreview]{
		Title:             "Preview Production Schedule Week Release",
		Method:            http.MethodGet,
		Route:             "/v1/operations/production-schedules/{id}/week-release-preview",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleWeekReleasePreview,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PreviewReleaseProductionScheduleWeekRequest) (*apiresource.ReleaseScheduleWeekPreview, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).PreviewReleaseProductionScheduleWeek
		},
	})
}
