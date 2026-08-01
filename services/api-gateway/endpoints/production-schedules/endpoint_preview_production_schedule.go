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

// Request to preview a production schedule.
type PreviewProductionScheduleRequest struct {
	// The instant to plan against. Defaults to now.
	PlanningAsOf field.Optional[time.Time] `json:"planning_as_of,omitzero"`
	// Overrides the configured horizon for this preview only.
	HorizonWeeks field.Optional[int32] `json:"horizon_weeks,omitzero" validate:"omitempty,min=1,max=104"`
	// Overrides the configured demand basis for this preview only.
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
// This is the inspection surface for the scheduler: it takes the same path a generated schedule will take, minus the write, so a plan can be reviewed and compared before anything depends on it. Machines must be marked as the planning constraint in production schedule settings, otherwise there is nothing to schedule.
type PreviewProductionScheduleEndpoint struct{}

func (e *PreviewProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*PreviewProductionScheduleRequest, *apiresource.ProductionSchedulePreview] {
	return (&apiendpoint.APIEndpoint[*PreviewProductionScheduleRequest, *apiresource.ProductionSchedulePreview]{
		Title:             "Preview Production Schedule",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/actions/preview",
		SuccessStatusCode: http.StatusOK,
		// Internal while the solver is being validated against the existing script.
		Public:     false,
		Preview:    true,
		AgentTool:  true,
		ObjectType: constants.ObjectTypeProductionSchedulePreview,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PreviewProductionScheduleRequest) (*apiresource.ProductionSchedulePreview, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).PreviewProductionSchedule
		},
	})
}
