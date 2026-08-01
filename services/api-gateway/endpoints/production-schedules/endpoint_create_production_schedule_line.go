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

// Request to add a campaign to a schedule by hand.
type CreateProductionScheduleLineRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Horizon week to plan the campaign in, zero-based.
	WeekIndex int32 `json:"week_index" validate:"gte=0"`
	// ID of the machine that will run it.
	MachineID string `json:"machine_id" validate:"required"`
	// ID of the item to build.
	ItemID string `json:"item_id" validate:"required"`
	// Units to build.
	Quantity float64 `json:"quantity" validate:"required,gt=0"`
	// Lots the quantity represents.
	Lots field.Optional[int32] `json:"lots,omitzero" validate:"omitempty,gte=0"`
	// Machine hours the campaign will take.
	RunHours field.Optional[float64] `json:"run_hours,omitzero" validate:"omitempty,gte=0"`
	// Why the campaign was added. Required when it lands inside a frozen week.
	Reason field.Optional[constants.ScheduleChangeReason] `json:"reason,omitzero"`
	// Free-form explanation of the change.
	ReasonNote field.Optional[string] `json:"reason_note,omitzero" validate:"omitempty,max=2000"`
}

var sampleCreateProductionScheduleLineRequest = &CreateProductionScheduleLineRequest{
	ProductionScheduleID: apiresource.SampleProductionScheduleID,
	WeekIndex:            2,
	MachineID:            apiresource.SampleMachineID,
	ItemID:               apiresource.SampleItemID,
	Quantity:             600,
}

func (*CreateProductionScheduleLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductionScheduleLineRequest)
}

// Adds a campaign to a schedule by hand.
//
// The line is recorded as manual, so a later regenerate can tell it apart from what the solver produced, and the change is written to the deviation log. Adding into a frozen week requires a `reason`.
type CreateProductionScheduleLineEndpoint struct{}

func (e *CreateProductionScheduleLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionScheduleLineRequest, *apiresource.ProductionScheduleLine] {
	return (&apiendpoint.APIEndpoint[*CreateProductionScheduleLineRequest, *apiresource.ProductionScheduleLine]{
		Title:             "Create Production Schedule Line",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/lines",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionScheduleLineRequest) (*apiresource.ProductionScheduleLine, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).CreateProductionScheduleLine
		},
	})
}
