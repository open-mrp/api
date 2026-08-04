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

// Request to re-solve a draft in place.
type RegenerateProductionScheduleRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// What happens to the campaigns someone placed or edited by hand.
	//
	// - `preserve_manual`: hand-edited campaigns are kept, and the fresh solve plans around them — their stock and machine time are facts the rest of the plan responds to.
	// - `replace_all`: hand edits are discarded and the fresh solve is taken whole.
	//
	// Omitting this keeps hand edits, because the alternative destroys work silently.
	MergeMode field.Optional[constants.ScheduleMergeMode] `json:"merge_mode,omitzero"`
	// The instant to plan against, which is what stock, demand history and active demand overrides are read as of.
	//
	// Defaults to now rather than to the instant the version was first generated, so a plain call answers "what would the solver say today". Because the horizon re-anchors to the week containing this instant, a kept campaign keeps the calendar week it was planned in but can end up under a different `week_index`.
	PlanningAsOf field.Optional[time.Time] `json:"planning_as_of,omitzero"`
	// Number of weeks the re-solve should cover, defaulting to the horizon this version already has.
	HorizonWeeks field.Optional[int32] `json:"horizon_weeks,omitzero" validate:"omitempty,gte=1,lte=104"`
	// How future demand is derived, defaulting to the basis this version was solved with.
	//
	// - `trailing_12`: demand is the trailing twelve months of orders.
	// - `seasonal_ema`: demand is a seasonal exponential moving average, which follows a season arriving early or late rather than flattening it.
	DemandBasis field.Optional[constants.ScheduleDemandBasis] `json:"demand_basis,omitzero"`
}

var sampleRegenerateProductionScheduleRequest = &RegenerateProductionScheduleRequest{
	ProductionScheduleID: apiresource.SampleProductionScheduleID,
	MergeMode:            field.Some(constants.ScheduleMergeModePreserveManual),
}

func (*RegenerateProductionScheduleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRegenerateProductionScheduleRequest)
}

// Re-solves a draft in place, keeping its version number.
//
// Only a draft can be regenerated. A published version is a commitment the floor is already working to, and a superseded or archived one is history; re-solving either in place would change what a week was measured against after the fact. To replan against a published version, generate a new one — publishing it supersedes the current one.
//
// The version number is kept deliberately: minting a new version for every re-solve would fill the list with drafts nobody asked for and make the version number meaningless as a count of the plans actually considered.
//
// Every hand edit a `replace_all` destroys is written to the deviation log before it goes, so "where did my change go" stays answerable. Call `preview-regenerate` first to see what a re-solve would change.
//
// Aside from the hand edits a `preserve_manual` run keeps, the version's campaigns, policy snapshot, derived department work, solver diagnostics and settings snapshot are all replaced with the fresh solve's, so the plan can still explain itself afterwards.
type RegenerateProductionScheduleEndpoint struct{}

func (e *RegenerateProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*RegenerateProductionScheduleRequest, *apiresource.ProductionSchedule] {
	return (&apiendpoint.APIEndpoint[*RegenerateProductionScheduleRequest, *apiresource.ProductionSchedule]{
		Title:             "Regenerate Production Schedule",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/actions/regenerate",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RegenerateProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).RegenerateProductionSchedule
		},
	})
}
