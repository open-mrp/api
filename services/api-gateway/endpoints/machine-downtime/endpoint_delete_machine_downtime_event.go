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

// Request to delete a machine downtime event.
type DeleteMachineDowntimeEventRequest struct {
	// ID of the downtime event to delete.
	MachineDowntimeEventID string `path:"id" validate:"required"`
}

// Deletes a machine downtime event.
//
// Meant for a stoppage that was logged by mistake: the event is removed permanently and stops counting against the machine's availability. To correct a real stoppage, update it instead so the record of the downtime survives.
type DeleteMachineDowntimeEventEndpoint struct{}

func (e *DeleteMachineDowntimeEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMachineDowntimeEventRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteMachineDowntimeEventRequest, *apiresource.EmptyResource]{
		Title:             "Delete Machine Downtime Event",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeMachineDowntimeEvent,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMachineDowntimeEventRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(MachineDowntimeSvc).DeleteDowntimeEvent
		},
	})
}
