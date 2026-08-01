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

// Request to log a machine downtime event.
type CreateMachineDowntimeEventRequest struct {
	// ID of the machine that stopped.
	MachineID string `json:"machine_id" validate:"required"`
	// Why the machine stopped.
	Reason constants.MachineDowntimeReasonCode `json:"reason" validate:"required"`
	// When the machine stopped.
	StartedAt time.Time `json:"started_at" validate:"required"`
	// When the machine started running again. Omit while the machine is still down.
	EndedAt field.Optional[time.Time] `json:"ended_at,omitzero"`
	// ID of the item the machine was running when it stopped.
	ItemID field.Optional[string] `json:"item_id,omitzero"`
	// ID of the production run in progress when the machine stopped.
	ProductionRunID field.Optional[string] `json:"production_run_id,omitzero"`
	// ID of the batch in progress when the machine stopped.
	BatchID field.Optional[string] `json:"batch_id,omitzero"`
	// Free-form notes about the stoppage.
	Note field.Optional[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
	// How the event was recorded.
	Source field.Optional[constants.MachineDowntimeSource] `json:"source,omitzero"`
}

var sampleCreateMachineDowntimeEventRequest = &CreateMachineDowntimeEventRequest{
	MachineID: apiresource.SampleMachineID,
	Reason:    constants.MachineDowntimeReasonCodeBreakdown,
	StartedAt: apiresource.SampleMachineDowntimeEvent.StartedAt,
}

func (*CreateMachineDowntimeEventRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMachineDowntimeEventRequest)
}

// Logs a machine downtime event.
//
// Omit `ended_at` while the machine is still down; a machine can only have one open event at a time. The department and production step are resolved from the machine, and the duration is calculated when the event is closed.
type CreateMachineDowntimeEventEndpoint struct{}

func (e *CreateMachineDowntimeEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent] {
	return (&apiendpoint.APIEndpoint[*CreateMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent]{
		Title:             "Create Machine Downtime Event",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeMachineDowntimeEvent,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError) {
			return svc.(MachineDowntimeSvc).CreateDowntimeEvent
		},
		LocationFunc: func(resp *apiresource.MachineDowntimeEvent) string {
			return "/v1/operations/machine-downtime-events/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachineDowntimeEvent,
			Fields:     []string{"machine", "department", "item", "reported_by"},
		}),
	})
}
