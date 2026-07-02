package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a machine.
type CreateMachineRequest struct {
	// Display name of the machine.
	//
	// Must be unique within your account; maximum 255 characters.
	Name string `json:"name" validate:"required,max=255"`
	// Serial number of the machine.
	//
	// Maximum 255 characters.
	SerialNumber string `json:"serial_number" validate:"required,max=255"`
	// Free-form notes about the machine.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// ID of the department this machine belongs to.
	//
	// Must reference a department in your account.
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

// Creates a machine and assigns it to a department.
//
// Returns a conflict error if a machine with the same name already exists.
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
		ObjectType:        constants.ObjectTypeMachine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionCreate},
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"department"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).CreateMachine
		},
		LocationFunc: func(resp *apiresource.Machine) string {
			return "/v1/operations/machines/" + resp.ID
		},
	})
}
