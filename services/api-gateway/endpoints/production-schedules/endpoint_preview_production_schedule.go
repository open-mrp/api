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

// Request to preview a production schedule.
type PreviewProductionScheduleRequest struct {
	// The instant to plan against, which is what stock, demand history and active demand overrides are read as of.
	//
	// Left unset, the preview is solved against the moment the request arrives. The horizon starts on the account's configured week-start day on or before this instant, so backdating this shifts the whole week grid.
	PlanningAsOf field.Optional[time.Time] `json:"planning_as_of,omitzero"`
	// Number of weeks the plan should cover, overriding the account's configured horizon for this preview only.
	HorizonWeeks field.Optional[int32] `json:"horizon_weeks,omitzero" validate:"omitempty,min=1,max=104"`
	// How future demand is derived, overriding the account's configured basis for this preview only.
	//
	// - `trailing_12`: demand is the trailing twelve months of orders.
	// - `seasonal_ema`: demand is a seasonal exponential moving average, which follows a season arriving early or late rather than flattening it.
	DemandBasis field.Optional[constants.ScheduleDemandBasis] `json:"demand_basis,omitzero"`
}

// Every field is optional: the common call is an empty body, which plans against now using the account's configured horizon and demand basis. The example overrides the horizon to show the shape.
var samplePreviewProductionScheduleRequest = &PreviewProductionScheduleRequest{
	HorizonWeeks: field.Some(int32(13)),
}

func (*PreviewProductionScheduleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePreviewProductionScheduleRequest)
}

// Runs the production scheduling solver and returns the plan without saving it.
//
// This is the inspection surface for the scheduler: it takes the same path a generated schedule will take, minus the write, so a plan can be reviewed and compared before anything depends on it. No version is created and nothing is numbered, so this can be called as often as needed.
//
// The solver plans the constraint department — the room that sets the pace of the factory — so production schedule settings must name one and it must have machines that are included in planning. Without that there is nothing to schedule and the request is rejected rather than returning an empty plan.
type PreviewProductionScheduleEndpoint struct{}

func (e *PreviewProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*PreviewProductionScheduleRequest, *apiresource.ProductionSchedulePreview] {
	return (&apiendpoint.APIEndpoint[*PreviewProductionScheduleRequest, *apiresource.ProductionSchedulePreview]{
		Title:             "Preview Production Schedule",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/actions/preview",
		SuccessStatusCode: http.StatusOK,
		// A preview writes nothing, so it asks for read rather than the create a generate needs.
		Public:     true,
		Preview:    true,
		AgentTool:  true,
		ReadOnly:   true,
		ObjectType: constants.ObjectTypeProductionSchedulePreview,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PreviewProductionScheduleRequest) (*apiresource.ProductionSchedulePreview, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).PreviewProductionSchedule
		},
	})
}
