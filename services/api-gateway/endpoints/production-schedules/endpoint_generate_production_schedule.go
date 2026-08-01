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

// Request to generate a production schedule.
type GenerateProductionScheduleRequest struct {
	// The instant to plan against. Defaults to now.
	PlanningAsOf field.Optional[time.Time] `json:"planning_as_of,omitzero"`
	// Overrides the configured horizon for this version only.
	HorizonWeeks field.Optional[int32] `json:"horizon_weeks,omitzero" validate:"omitempty,min=1,max=104"`
	// Overrides the configured demand basis for this version only.
	DemandBasis field.Optional[constants.ScheduleDemandBasis] `json:"demand_basis,omitzero"`
	// Label for the version.
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
// The plan is saved as a draft: nothing is frozen and every line is editable until the version is published. Generating again creates a new version rather than replacing this one, because attainment is measured against whichever version was live at the time.
type GenerateProductionScheduleEndpoint struct{}

func (e *GenerateProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*GenerateProductionScheduleRequest, *apiresource.ProductionSchedule] {
	return (&apiendpoint.APIEndpoint[*GenerateProductionScheduleRequest, *apiresource.ProductionSchedule]{
		Title:             "Generate Production Schedule",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
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
