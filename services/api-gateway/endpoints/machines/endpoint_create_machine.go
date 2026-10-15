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

// Request to create a machine.
type CreateMachineRequest struct {
	// Display name of the machine.
	//
	// Unique within your account (case-insensitive); maximum 255 characters.
	Name string `json:"name" validate:"required,max=255"`
	// Serial number of the machine.
	//
	// Unique within your account (case-insensitive); maximum 255 characters.
	SerialNumber string `json:"serial_number" validate:"required,max=255"`
	// ID of the work center the machine belongs to.
	WorkCenterID string `json:"work_center_id" validate:"required"`
	// `active` or `archived`. Defaults to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleCreateMachineRequest = &CreateMachineRequest{
	Name:         "CNC Router",
	SerialNumber: "SN-2024-0001",
	WorkCenterID: apiresource.SampleWorkCenterID,
}

func (*CreateMachineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMachineRequest)
}

// Creates a machine at a work center.
//
// Returns a conflict error if another machine already has the name or serial number.
type CreateMachineEndpoint struct{}

func (e *CreateMachineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMachineRequest, *apiresource.Machine] {
	return (&apiendpoint.APIEndpoint[*CreateMachineRequest, *apiresource.Machine]{
		Title:             "Create Machine",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeMachine,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionCreate},
		}},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachine,
			Fields:     []string{"work_center"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMachineRequest) (*apiresource.Machine, *apierror.APIError) {
			return svc.(MachineSvc).CreateMachine
		},
		LocationFunc: func(resp *apiresource.Machine) string {
			return "/v1/operations/machines/" + resp.ID
		},
	})
}
