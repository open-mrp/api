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
type RetrieveMachineRequest struct {
	// Machine ID.
	MachineID string `path:"id" validate:"required"`
}

type RetrieveMachineEndpoint struct{}

func (e *RetrieveMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMachineRequest, *apiresource.Machine] {
	return &apiendpoint.APIEndpoint[*RetrieveMachineRequest, *apiresource.Machine]{
		Title:             "Retrieve Machine",
		Description:       "Returns a machine by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		Request:           &RetrieveMachineRequest{},
		Response:          &apiresource.Machine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).GetMachine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"department"},
		}),
	}
}
