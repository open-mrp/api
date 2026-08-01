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
	// Date to plan from. Defaults to the date the version was generated for.
	PlanningAsOf field.Optional[time.Time] `json:"planning_as_of,omitzero"`
	// Weeks the plan should cover. Defaults to the version's own horizon.
	HorizonWeeks field.Optional[int32] `json:"horizon_weeks,omitzero" validate:"omitempty,gte=1,lte=104"`
	// How demand is derived. Defaults to the version's own basis.
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
// Every campaign either plan holds is listed, including the ones both agree on, so the caller can render a full side-by-side rather than a list of surprises. `discarded_manual_count` is what `replace_all` would destroy — a regenerate that silently eats hand-work is abandoned within two cycles, so the destructive mode has to be able to state its cost before it runs.
//
// Planning inputs default to the ones the version was generated with, so a plain preview answers "what would the solver say now" rather than answering a different question with a different horizon.
type PreviewRegenerateProductionScheduleEndpoint struct{}

func (e *PreviewRegenerateProductionScheduleEndpoint) Materialize() *apiendpoint.APIEndpoint[*PreviewRegenerateProductionScheduleRequest, *apiresource.ProductionScheduleRegeneratePreview] {
	return (&apiendpoint.APIEndpoint[*PreviewRegenerateProductionScheduleRequest, *apiresource.ProductionScheduleRegeneratePreview]{
		Title:             "Preview Production Schedule Regenerate",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/actions/preview-regenerate",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
