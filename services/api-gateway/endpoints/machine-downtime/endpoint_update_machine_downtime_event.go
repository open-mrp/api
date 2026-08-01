package machinedowntimeep

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

// Request to update a machine downtime event.
type UpdateMachineDowntimeEventRequest struct {
	// ID of the downtime event.
	MachineDowntimeEventID string `path:"id" validate:"required"`
	// Why the machine stopped.
	Reason field.Optional[constants.MachineDowntimeReasonCode] `json:"reason,omitzero"`
	// When the machine stopped.
	StartedAt field.Optional[time.Time] `json:"started_at,omitzero"`
	// When the machine started running again. Send null to reopen an event that was closed by mistake.
	EndedAt field.Clearable[time.Time] `json:"ended_at,omitzero"`
	// ID of the item the machine was running when it stopped. Send null to detach the item.
	ItemID field.Clearable[string] `json:"item_id,omitzero"`
	// ID of the production run in progress when the machine stopped. Send null to detach the run.
	ProductionRunID field.Clearable[string] `json:"production_run_id,omitzero"`
	// ID of the batch in progress when the machine stopped. Send null to detach the batch.
	BatchID field.Clearable[string] `json:"batch_id,omitzero"`
	// Free-form notes about the stoppage. Send null to remove the note.
	Note field.Clearable[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
}

var sampleUpdateMachineDowntimeEventRequest = &UpdateMachineDowntimeEventRequest{
	MachineDowntimeEventID: apiresource.SampleMachineDowntimeEventID,
	EndedAt:                field.Set(*apiresource.SampleMachineDowntimeEvent.EndedAt),
}

func (*UpdateMachineDowntimeEventRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMachineDowntimeEventRequest)
}

// Updates a machine downtime event.
//
// Setting `ended_at` closes the event and calculates its duration.
type UpdateMachineDowntimeEventEndpoint struct{}

func (e *UpdateMachineDowntimeEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent] {
	return (&apiendpoint.APIEndpoint[*UpdateMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent]{
		Title:             "Update Machine Downtime Event",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
