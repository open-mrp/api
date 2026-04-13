package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a machine.
type UpdateMachineRequest struct {
	// Machine ID.
	MachineID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Serial number.
	SerialNumber *string `json:"serial_number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Notes.
	Notes *string `json:"notes,omitempty" nullable:"false"`
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
		ContentType:       "application/json",
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
