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
	// ID of the downtime event.
	MachineDowntimeEventID string `path:"id" validate:"required"`
}

// Deletes a machine downtime event.
type DeleteMachineDowntimeEventEndpoint struct{}

func (e *DeleteMachineDowntimeEventEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMachineDowntimeEventRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteMachineDowntimeEventRequest, *apiresource.EmptyResource]{
		Title:             "Delete Machine Downtime Event",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/machine-downtime-events/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
