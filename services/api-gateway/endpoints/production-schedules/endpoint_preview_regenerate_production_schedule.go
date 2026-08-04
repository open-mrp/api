package productionscheduleep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to see what a re-solve would change.
type PreviewRegenerateProductionScheduleRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// The instant to plan against, which is what stock, demand history and active demand overrides are read as of.
	//
	// Defaults to now rather than to the instant the version was first generated, so a plain call answers "what would the solver say today". Because the horizon re-anchors to the week containing this instant, a campaign can appear under a different `week_index` than the one stored on the draft.
	PlanningAsOf field.Optional[time.Time] `json:"planning_as_of,omitzero"`
	// Number of weeks the re-solve should cover, defaulting to the horizon this version already has.
	HorizonWeeks field.Optional[int32] `json:"horizon_weeks,omitzero" validate:"omitempty,gte=1,lte=104"`
	// How future demand is derived, defaulting to the basis this version was solved with.
	//
	// - `trailing_12`: demand is the trailing twelve months of orders.
	// - `seasonal_ema`: demand is a seasonal exponential moving average, which follows a season arriving early or late rather than flattening it.
	DemandBasis field.Optional[constants.ScheduleDemandBasis] `json:"demand_basis,omitzero"`
}

var samplePreviewRegenerateProductionScheduleRequest = &PreviewRegenerateProductionScheduleRequest{
	ProductionScheduleID: apiresource.SampleProductionScheduleID,
	HorizonWeeks:         field.Some(int32(13)),
}

func (*PreviewRegenerateProductionScheduleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePreviewRegenerateProductionScheduleRequest)
}

// Returns what regenerating this draft would change, without changing it.
//
// Every campaign either plan holds is listed, including the ones both agree on, so the caller can render a full side-by-side rather than a list of surprises. Only a draft can be previewed, for the same reason only a draft can be regenerated.
//
// The comparison is run the way a regenerate runs by default — hand-edited campaigns are kept, and the fresh solve plans around them — so they read as unchanged rather than as work the solver wants to take away. `manual_line_count` is how many campaigns on the draft were placed or edited by hand, which is the work a `replace_all` regenerate is putting at risk.
//
// The horizon and demand basis default to the ones this version already has, so a plain call changes only how current the plan is, not what question is being asked.
type PreviewRegenerateProductionScheduleEndpoint struct{}

func (e *PreviewRegenerateProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*PreviewRegenerateProductionScheduleRequest, *apiresource.ProductionScheduleRegeneratePreview] {
	return (&apiendpoint.APIEndpoint[*PreviewRegenerateProductionScheduleRequest, *apiresource.ProductionScheduleRegeneratePreview]{
		Title:             "Preview Production Schedule Regenerate",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/actions/preview-regenerate",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionScheduleRegeneratePreview,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PreviewRegenerateProductionScheduleRequest) (*apiresource.ProductionScheduleRegeneratePreview, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).PreviewRegenerateProductionSchedule
		},
	})
}
