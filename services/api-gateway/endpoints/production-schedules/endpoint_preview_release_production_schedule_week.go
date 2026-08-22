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

// Request to preview releasing one week of a production schedule.
type PreviewReleaseProductionScheduleWeekRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Zero-based week offset from the start of the horizon.
	WeekIndex int32 `query:"week_index" validate:"min=0"`
	// Preview the week as if every batch were newly issued.
	//
	// By default the preview counts tickets an earlier week issued and the floor never worked against this week's campaigns, because that is what releasing would do.
	SkipCarryForward bool `query:"skip_carry_forward"`
}

// Returns what releasing a week would create, without creating it.
//
// The lots are resolved exactly as the release itself resolves them, so what a planner is shown and what the floor receives cannot drift apart.
//
// `is_releasable` is false when the week is empty or already released, with `blocked_reason` saying which; `existing_production_run_id` names the run a released week is already tied to.
//
// Cancelled campaigns and campaigns planned at zero are excluded here exactly as the release excludes them, so a week holding nothing but those previews as empty.
//
// Lots the floor is already holding are named as such. A batch with `carried_forward_from` set is a ticket an earlier week issued and nobody worked, which the release moves into the new run rather than reissuing, so nothing has to be reprinted.
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
