package machinedowntimeep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeMachineDowntimeEvent,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachineDowntime, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMachineDowntimeEventRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(MachineDowntimeSvc).DeleteDowntimeEvent
		},
	})
}
