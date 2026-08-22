package machinedowntimeep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a machine downtime event.
type UpdateMachineDowntimeEventRequest struct {
	// ID of the downtime event to update.
	MachineDowntimeEventID string `path:"id" validate:"required"`
	// ID of the machine that stopped.
	//
	// Moving an event to another machine re-resolves the department it is charged to, so past availability changes for both rooms. Rejected when the destination machine already has an open stoppage and this one is open too.
	MachineID field.Optional[string] `json:"machine_id,omitzero"`
	// Why the machine stopped.
	//
	// Reclassifying a stoppage moves it to the OEE term the new reason charges, so past availability figures change with it.
	Reason field.Optional[constants.MachineDowntimeReasonCode] `json:"reason,omitzero"`
	// When the machine stopped.
	//
	// Correcting it recalculates the duration and can move the stoppage onto a different business day.
	StartedAt field.Optional[time.Time] `json:"started_at,omitzero"`
	// When the machine started running again.
	//
	// Setting it closes the event and records the duration. Send null to reopen an event that was closed by mistake, which is rejected if the machine has since had another stoppage logged that is still open.
	EndedAt field.Clearable[time.Time] `json:"ended_at,omitzero"`
	// How long the machine was down, counted in a unit of time.
	//
	// Restates the end as a length of time from the start, so it is applied against `started_at` as this request leaves it. Send this or `ended_at`, never both. Send null to reopen the event.
	Duration field.Clearable[apirequest.QuantityInput] `json:"duration,omitzero"`
	// ID of the item the machine was running when it stopped.
	//
	// Send null to detach the item.
	ItemID field.Clearable[string] `json:"item_id,omitzero"`
	// ID of the production run in progress when the machine stopped.
	//
	// Send null to detach the run.
	ProductionRunID field.Clearable[string] `json:"production_run_id,omitzero"`
	// ID of the batch in progress when the machine stopped.
	//
	// Send null to detach the batch.
	BatchID field.Clearable[string] `json:"batch_id,omitzero"`
	// Free-form notes about the stoppage.
	//
	// Send null to remove the note. Maximum 2000 characters.
	Note field.Clearable[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
}

var sampleUpdateMachineDowntimeEventRequest = &UpdateMachineDowntimeEventRequest{
	MachineDowntimeEventID: apiresource.SampleMachineDowntimeEventID,
	EndedAt:                field.Set(*apiresource.SampleMachineDowntimeEvent.EndedAt),
}

func (*UpdateMachineDowntimeEventRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMachineDowntimeEventRequest)
}

// Closes or corrects a machine downtime event.
//
// Only the fields provided in the request are changed. Setting `ended_at` — or a `duration`, which says the same thing as a length of time from the start — closes the event and calculates how long it lasted; sending either as null reopens an event closed by mistake, which is rejected when the machine already has another open stoppage. Moving the event to another machine re-resolves the department the stoppage is charged to.
type UpdateMachineDowntimeEventEndpoint struct{}

func (e *UpdateMachineDowntimeEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent] {
	return (&apiendpoint.APIEndpoint[*UpdateMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent]{
		Title:             "Update Machine Downtime Event",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeMachineDowntimeEvent,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError) {
			return svc.(MachineDowntimeSvc).UpdateDowntimeEvent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachineDowntimeEvent,
			Fields:     []string{"machine", "department", "item", "reported_by"},
		}),
	})
}
