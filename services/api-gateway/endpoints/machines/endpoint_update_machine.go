package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateMachineRequest is the request to partially update a machine.
type UpdateMachineRequest struct {
	// The ID of the machine to update.
	MachineID string `path:"id" validate:"required"`
	// The display name of the machine.
	Name *string `json:"name,omitempty"`
	// The serial number of the machine.
	SerialNumber *string `json:"serial_number,omitempty"`
	// Optional notes about the machine.
	Notes *string `json:"notes,omitempty"`
}

var sampleUpdateMachineName = "Updated CNC Router"
var sampleUpdateMachineRequest = &UpdateMachineRequest{
	Name: &sampleUpdateMachineName,
}

func (*UpdateMachineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMachineRequest)
}

type UpdateMachineEndpoint struct{}

func (e *UpdateMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMachineRequest, *apiresource.Machine] {
	return &apiendpoint.APIEndpoint[*UpdateMachineRequest, *apiresource.Machine]{
		Title:             "Update Machine",
		Description:       "Partially updates a machine.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/machines/{id}",
		Request:           &UpdateMachineRequest{},
		Response:          &apiresource.Machine{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).UpdateMachine
		},
	}
}
