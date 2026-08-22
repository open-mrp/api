package productionscheduleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to add a campaign to a schedule by hand.
type CreateProductionScheduleLineRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Horizon week to plan the campaign in, zero-based.
	//
	// Week 0 is the week the schedule's horizon starts in. The week must fall inside the horizon this version was planned over.
	WeekIndex int32 `json:"week_index" validate:"gte=0"`
	// ID of the machine that will run the campaign.
	//
	// The machine's production step and department are copied onto the campaign, which is what department-level attainment rolls it up by. The schedule's derived department work is not re-exploded for a hand-added campaign; it is rebuilt the next time the version is regenerated.
	MachineID string `json:"machine_id" validate:"required"`
	// ID of the item to build.
	ItemID string `json:"item_id" validate:"required"`
	// Units to build over the campaign.
	Quantity float64 `json:"quantity" validate:"required,gt=0"`
	// How many lots the quantity is built in.
	//
	// Left unset, it is derived from the quantity and the account's default lot size. The lot size itself is taken from that account default and is not settable per campaign, so this is a record of the lot count rather than what a release splits batches by.
	Lots field.Optional[int32] `json:"lots,omitzero" validate:"omitempty,gte=0"`
	// Machine hours the campaign will take.
	//
	// Left unset, it is estimated from the rate this version was solved with for this item, so the week's utilisation still reflects the added work. An item the version holds no policy for estimates to zero.
	RunHours field.Optional[float64] `json:"run_hours,omitzero" validate:"omitempty,gte=0"`
	// Why the campaign was added.
	//
	// Required when the campaign lands inside a frozen week, since that is a commitment being changed.
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
//
// Only a draft or a published version can be edited; a superseded or archived version is history. The campaign is appended to the end of its week's run order.
type CreateProductionScheduleLineEndpoint struct{}

func (e *CreateProductionScheduleLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionScheduleLineRequest, *apiresource.ProductionScheduleLine] {
	return (&apiendpoint.APIEndpoint[*CreateProductionScheduleLineRequest, *apiresource.ProductionScheduleLine]{
		Title:             "Create Production Schedule Line",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/lines",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
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
