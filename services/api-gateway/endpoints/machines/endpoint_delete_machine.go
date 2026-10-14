package machineep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMachineRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteMachineRequest, *object.Deleted]{
		Title:             "Delete Machine",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeMachine,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMachineRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(MachineSvc).DeleteMachine, constants.ObjectTypeMachine, "id")
		},
	})
}
