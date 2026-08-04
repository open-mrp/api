package productionscheduleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to edit a campaign on a schedule.
type UpdateProductionScheduleLineRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// ID of the schedule line.
	LineID string `path:"line_id" validate:"required"`
	// Horizon week to move the campaign to, zero-based.
	//
	// Must fall inside the horizon this version was planned over.
	WeekIndex field.Optional[int32] `json:"week_index,omitzero" validate:"omitempty,gte=0"`
	// ID of the machine to move the campaign to.
	MachineID field.Optional[string] `json:"machine_id,omitzero"`
	// Units to build over the campaign.
	//
	// Changing this does not re-derive `lots` or `run_hours` — send those alongside it when they should follow, or the campaign will keep claiming its old share of machine time.
	Quantity field.Optional[float64] `json:"quantity,omitzero" validate:"omitempty,gte=0"`
	// How many lots the quantity is built in.
	//
	// What a release actually splits batches by is the lot size the campaign was planned at, which this does not change.
	Lots field.Optional[int32] `json:"lots,omitzero" validate:"omitempty,gte=0"`
	// Machine hours the campaign will take.
	RunHours field.Optional[float64] `json:"run_hours,omitzero" validate:"omitempty,gte=0"`
	// Position within the week's run order, lowest first.
	SequenceIndex field.Optional[int32] `json:"sequence_index,omitzero" validate:"omitempty,gte=0"`
	// Progress of the campaign.
	//
	// Setting `released` here only labels the campaign; it does not create a production run or any batches — releasing a week to the floor is its own action. Setting `cancelled` leaves the campaign on the plan but excludes it from any later release of its week.
	Status field.Optional[constants.ProductionScheduleLineStatus] `json:"status,omitzero"`
	// Why the campaign changed.
	//
	// Required when the change touches a frozen week, including moving a campaign out of one.
	Reason field.Clearable[constants.ScheduleChangeReason] `json:"reason,omitzero"`
	// Free-form explanation of the change.
	ReasonNote field.Optional[string] `json:"reason_note,omitzero" validate:"omitempty,max=2000"`
}

var sampleUpdateProductionScheduleLineRequest = &UpdateProductionScheduleLineRequest{
	ProductionScheduleID: apiresource.SampleProductionScheduleID,
	LineID:               apiresource.SampleProductionScheduleLineID,
	Quantity:             field.Some(900.0),
}

func (*UpdateProductionScheduleLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductionScheduleLineRequest)
}

// Edits a campaign on a schedule.
//
// Every change is written to the deviation log with a full before-and-after snapshot, and the line becomes manual so a regenerate can tell it apart from solver output. A change that touches a frozen week — including moving a campaign out of one — requires a `reason`.
//
// Only a draft or a published version can be edited; a superseded or archived version is history. An edit that changes several things at once is logged under the single most significant one, in the order machine, week, quantity, position — that being the change a planner has to react to first.
type UpdateProductionScheduleLineEndpoint struct{}

func (e *UpdateProductionScheduleLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionScheduleLineRequest, *apiresource.ProductionScheduleLine] {
	return (&apiendpoint.APIEndpoint[*UpdateProductionScheduleLineRequest, *apiresource.ProductionScheduleLine]{
		Title:             "Update Production Schedule Line",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionScheduleLineRequest) (*apiresource.ProductionScheduleLine, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).UpdateProductionScheduleLine
		},
	})
}
