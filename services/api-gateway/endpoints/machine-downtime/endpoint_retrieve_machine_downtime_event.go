package machinedowntimeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a machine downtime event.
type RetrieveMachineDowntimeEventRequest struct {
	// ID of the downtime event to retrieve.
	MachineDowntimeEventID string `path:"id" validate:"required"`
}

// Returns a single machine downtime event.
type RetrieveMachineDowntimeEventEndpoint struct{}

func (e *RetrieveMachineDowntimeEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent] {
	return (&apiendpoint.APIEndpoint[*RetrieveMachineDowntimeEventRequest, *apiresource.MachineDowntimeEvent]{
		Title:             "Retrieve Machine Downtime Event",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeMachineDowntimeEvent,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMachineDowntimeEventRequest) (*apiresource.MachineDowntimeEvent, *apierror.APIError) {
			return svc.(MachineDowntimeSvc).GetDowntimeEvent
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachineDowntimeEvent,
			Fields:     []string{"machine", "department", "item", "reported_by"},
		}),
	})
}
