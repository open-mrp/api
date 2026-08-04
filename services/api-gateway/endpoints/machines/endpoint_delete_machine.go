package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a machine.
type DeleteMachineRequest struct {
	// ID of the machine to delete.
	MachineID string `path:"id" validate:"required"`
}

// Deletes a machine.
//
// Deletion is permanent, and repeating the call reports that the machine has already been deleted. Downtime events and schedule lines already logged against the machine are kept rather than removed with it.
type DeleteMachineEndpoint struct{}

func (e *DeleteMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMachineRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteMachineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Machine",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMachineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(MachineSvc).DeleteMachine
		},
	})
}
