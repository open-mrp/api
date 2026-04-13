package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a machine.
type GetMachineRequest struct {
	// Machine ID.
	MachineID string `path:"id" validate:"required"`
}

type GetMachineEndpoint struct{}

func (e *GetMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetMachineRequest, *apiresource.Machine] {
	return &apiendpoint.APIEndpoint[*GetMachineRequest, *apiresource.Machine]{
		Title:             "Get Machine",
		Description:       "Returns a machine by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		Request:           &GetMachineRequest{},
		Response:          &apiresource.Machine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).GetMachine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"department"},
		}),
	}
}
