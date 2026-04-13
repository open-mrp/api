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

type DeleteMachineEndpoint struct{}

func (e *DeleteMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMachineRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteMachineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Machine",
		Description:       "Deletes a machine.",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		Request:           &DeleteMachineRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMachineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(MachineSvc).DeleteMachine
		},
	}
}
