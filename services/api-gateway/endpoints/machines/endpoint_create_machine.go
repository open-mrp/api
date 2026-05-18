package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a machine.
type CreateMachineRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Serial number.
	SerialNumber string `json:"serial_number" validate:"required,max=255"`
	// Notes.
	Notes *string `json:"notes,omitempty"`
	// Department ID.
	DepartmentID string `json:"department_id" validate:"required"`
}

var sampleCreateMachineRequest = &CreateMachineRequest{
	Name:         "CNC Router",
	SerialNumber: "SN-2024-0001",
	DepartmentID: apiresource.SampleDepartmentID,
}

func (*CreateMachineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMachineRequest)
}

// Creates a machine and associates it with a department.
type CreateMachineEndpoint struct{}

func (e *CreateMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMachineRequest, *apiresource.Machine] {
	return (&apiendpoint.APIEndpoint[*CreateMachineRequest, *apiresource.Machine]{
		Title:             "Create Machine",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).CreateMachine
		},
		LocationFunc: func(resp *apiresource.Machine) string {
			return "/v1/operations/machines/" + resp.ID
		},
	})
}
