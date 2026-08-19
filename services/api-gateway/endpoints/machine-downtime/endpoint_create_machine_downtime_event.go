package machinedowntimeep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
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
	//
	// The reason decides which OEE term the stoppage charges, so it does more than label the event. Retrieve the available reasons and the term each one charges from the downtime reasons list.
	Reason constants.MachineDowntimeReasonCode `json:"reason" validate:"required"`
	// When the machine stopped.
	//
	// Cannot be in the future beyond a few minutes of clock skew, which is allowed so a shop-floor tablet running fast can still log "just now". The business day the stoppage counts against is taken from this timestamp.
	StartedAt time.Time `json:"started_at" validate:"required"`
	// When the machine started running again.
	//
	// Omit it while the machine is still down; that leaves the event open, and the duration is filled in once the event is closed. It must be later than `started_at`.
	EndedAt field.Optional[time.Time] `json:"ended_at,omitzero"`
	// How long the machine was down, counted in a unit of time.
	//
	// The end time is derived from `started_at` plus this. Send either send `ended_at` or `duration`. The unit must measure time.
	Duration field.Optional[apirequest.QuantityInput] `json:"duration,omitzero"`
	// ID of the item the machine was running when it stopped.
	ItemID field.Optional[string] `json:"item_id,omitzero"`
	// ID of the production run in progress when the machine stopped.
	ProductionRunID field.Optional[string] `json:"production_run_id,omitzero"`
	// ID of the batch in progress when the machine stopped.
	BatchID field.Optional[string] `json:"batch_id,omitzero"`
	// Free-form notes about the stoppage.
	//
	// Searchable from the downtime events list. Maximum 2000 characters.
	Note field.Optional[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
	// How the event was recorded.
	//
	// Records the stoppage as manually logged unless you say otherwise, so an integration or shop-floor station should send its own source to keep hand-entered downtime distinguishable.
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
// Give the stoppage an end either as `ended_at` or as a `duration` counted in a unit of time — sending both is rejected. Omit `ended_at` while the machine is still down. A machine can only have one open event at a time, so logging a second open stoppage against a machine that is already down is rejected until the first is closed.
//
// The department is taken from the machine, the business day is taken from `started_at`, the event is attributed to the credentials that made the request, and the duration is calculated when the event is closed.
type CreateMachineDowntimeEventEndpoint struct{}

func (e *CreateMachineDowntimeEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent] {
	return (&apiendpoint.APIEndpoint[*CreateMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent]{
		Title:             "Create Machine Downtime Event",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
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
