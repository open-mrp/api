package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateMachineRequest is the request to create a new machine.
type CreateMachineRequest struct {
	// The display name of the machine.
	Name string `json:"name" validate:"required"`
	// The serial number of the machine.
	SerialNumber string `json:"serial_number" validate:"required"`
	// Optional notes about the machine.
	Notes *string `json:"notes,omitempty"`
	// The ID of the department this machine belongs to.
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

type CreateMachineEndpoint struct{}

func (e *CreateMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMachineRequest, *apiresource.Machine] {
	return &apiendpoint.APIEndpoint[*CreateMachineRequest, *apiresource.Machine]{
		Title:             "Create Machine",
		Description:       "Creates a new machine and associates it with a department.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/machines",
		Request:           &CreateMachineRequest{},
		Response:          &apiresource.Machine{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).CreateMachine
		},
	}
}
