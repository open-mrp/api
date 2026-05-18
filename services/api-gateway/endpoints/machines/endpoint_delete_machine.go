package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a machine.
type DeleteMachineRequest struct {
	// Machine ID.
	MachineID string `path:"id" validate:"required"`
}

// Deletes a machine.
type DeleteMachineEndpoint struct{}

func (e *DeleteMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMachineRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteMachineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Machine",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMachineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(MachineSvc).DeleteMachine
		},
	})
}
