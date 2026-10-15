package machineep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to partially update a machine.
type UpdateMachineRequest struct {
	// ID of the machine to update.
	MachineID string `path:"id" validate:"required"`
	// Display name of the machine.
	//
	// Unique within your account (case-insensitive); maximum 255 characters.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Serial number of the machine.
	//
	// Unique within your account (case-insensitive); maximum 255 characters.
	SerialNumber field.Optional[string] `json:"serial_number,omitzero" validate:"omitempty,max=255"`
	// ID of the work center to move the machine to.
	WorkCenterID field.Optional[string] `json:"work_center_id,omitzero" validate:"omitempty"`
	// `active` or `archived`. An archived machine stays on the records that already name it but can't be newly assigned.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleUpdateMachineName = "Updated CNC Router"
var sampleUpdateMachineRequest = &UpdateMachineRequest{
	Name: field.Some(sampleUpdateMachineName),
}

func (*UpdateMachineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMachineRequest)
}

// Updates a machine.
//
// Only the fields you send change. Returns a conflict error if another machine already has the new name or serial number.
type UpdateMachineEndpoint struct{}

func (e *UpdateMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMachineRequest, *apiresource.Machine] {
	return (&apiendpoint.APIEndpoint[*UpdateMachineRequest, *apiresource.Machine]{
		Title:             "Update Machine",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeMachine,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionUpdate},
		}},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"work_center"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).UpdateMachine
		},
	})
}
