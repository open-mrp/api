package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a machine.
type UpdateMachineRequest struct {
	// Machine ID.
	MachineID string `path:"id" validate:"required"`
	// Display name of the machine.
	//
	// Must be unique within your account; maximum 255 characters.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Serial number of the machine.
	//
	// Maximum 255 characters.
	SerialNumber field.Optional[string] `json:"serial_number,omitzero" validate:"omitempty,max=255"`
	// Free-form notes about the machine.
	Notes field.Optional[string] `json:"notes,omitzero"`
}

var sampleUpdateMachineName = "Updated CNC Router"
var sampleUpdateMachineRequest = &UpdateMachineRequest{
	Name: field.Some(sampleUpdateMachineName),
}

func (*UpdateMachineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMachineRequest)
}

// Partially updates a machine.
//
// Only the fields provided in the request are changed. Returns a conflict error if the new name is already in use by another machine.
type UpdateMachineEndpoint struct{}

func (e *UpdateMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMachineRequest, *apiresource.Machine] {
	return (&apiendpoint.APIEndpoint[*UpdateMachineRequest, *apiresource.Machine]{
		Title:             "Update Machine",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeMachine,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"department"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).UpdateMachine
		},
	})
}
