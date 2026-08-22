package productionscheduleep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to generate a production schedule.
type GenerateProductionScheduleRequest struct {
	// The instant to plan against, which is what stock, demand history and active demand overrides are read as of.
	//
	// Left unset, the plan is solved against the moment the request arrives. The horizon starts on the account's configured week-start day on or before this instant, so backdating this shifts the whole week grid.
	PlanningAsOf field.Optional[time.Time] `json:"planning_as_of,omitzero"`
	// Number of weeks the plan should cover, overriding the account's configured horizon for this version only.
	HorizonWeeks field.Optional[int32] `json:"horizon_weeks,omitzero" validate:"omitempty,min=1,max=104"`
	// How future demand is derived, overriding the account's configured basis for this version only.
	//
	// - `trailing_12`: demand is the trailing twelve months of orders.
	// - `seasonal_ema`: demand is a seasonal exponential moving average, which follows a season arriving early or late rather than flattening it.
	DemandBasis field.Optional[constants.ScheduleDemandBasis] `json:"demand_basis,omitzero"`
	// Human-readable label for the version, such as the week it was cut for.
	//
	// Purely for recognising the version in a list; versions are numbered automatically and the number is what identifies them.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
}

var sampleGenerateProductionScheduleRequest = &GenerateProductionScheduleRequest{
	Name: field.Some("2026-W31 knit plan"),
}

func (*GenerateProductionScheduleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleGenerateProductionScheduleRequest)
}

// Generates and saves a new production schedule.
//
// The plan is saved as a draft: nothing is frozen yet, so campaigns can be added, changed and removed without having to give a reason. Generating again creates a new version rather than replacing this one, because attainment is measured against whichever version was live at the time.
//
// The solver plans the constraint department — the room that sets the pace of the factory — so production schedule settings must name one and it must have machines that are included in planning. Without that there is nothing to schedule and the request is rejected rather than returning an empty plan.
//
// Alongside the campaigns, the version stores the assumptions it was solved with, the per-item policies behind each campaign, and the downstream department work implied by the plan.
type GenerateProductionScheduleEndpoint struct{}

func (e *GenerateProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*GenerateProductionScheduleRequest, *apiresource.ProductionSchedule] {
	return (&apiendpoint.APIEndpoint[*GenerateProductionScheduleRequest, *apiresource.ProductionSchedule]{
		Title:             "Generate Production Schedule",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeProductionSchedule,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GenerateProductionScheduleRequest) (*apiresource.ProductionSchedule, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).GenerateProductionSchedule
		},
		LocationFunc: func(resp *apiresource.ProductionSchedule) string {
			return "/v1/operations/production-schedules/" + resp.ID
		},
	})
}
