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
	// What happens to hand-edited campaigns. Defaults to keeping them.
	MergeMode field.Optional[constants.ScheduleMergeMode] `json:"merge_mode,omitzero"`
	// Date to plan from. Defaults to the date the version was generated for.
	PlanningAsOf field.Optional[time.Time] `json:"planning_as_of,omitzero"`
	// Weeks the plan should cover. Defaults to the version's own horizon.
	HorizonWeeks field.Optional[int32] `json:"horizon_weeks,omitzero" validate:"omitempty,gte=1,lte=104"`
	// How demand is derived. Defaults to the version's own basis.
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
// `preserve_manual` keeps every hand-edited campaign and replaces the rest. `replace_all` takes the fresh solve whole, and each hand edit it destroys is written to the deviation log first — "where did my change go" has to stay answerable. Call `preview-regenerate` first to see the cost as a number.
type RegenerateProductionScheduleEndpoint struct{}

func (e *RegenerateProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*RegenerateProductionScheduleRequest, *apiresource.ProductionSchedule] {
	return (&apiendpoint.APIEndpoint[*RegenerateProductionScheduleRequest, *apiresource.ProductionSchedule]{
		Title:             "Regenerate Production Schedule",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/actions/regenerate",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
