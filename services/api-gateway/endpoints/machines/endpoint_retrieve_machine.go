package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a machine.
type RetrieveMachineRequest struct {
	// ID of the machine to retrieve.
	MachineID string `path:"id" validate:"required"`
}

// Returns a machine by ID.
type RetrieveMachineEndpoint struct{}

func (e *RetrieveMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMachineRequest, *apiresource.Machine] {
	return (&apiendpoint.APIEndpoint[*RetrieveMachineRequest, *apiresource.Machine]{
		Title:             "Retrieve Machine",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeMachine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).GetMachine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"department"},
		}),
	})
}
