package departmentep

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

// Request to partially update a department.
type UpdateDepartmentRequest struct {
	// Department ID.
	DepartmentID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Notes about the department.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Storage location ID.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// Scanning station IDs to connect (additive).
	ScanningStationIDs []string `json:"scanning_station_ids,omitzero"`
	// Machine IDs to connect (additive).
	MachineIDs []string `json:"machine_ids,omitzero"`
}

var sampleUpdateDepartmentName = "Production"
var sampleUpdateDepartmentRequest = &UpdateDepartmentRequest{
	Name: field.Some(sampleUpdateDepartmentName),
}

func (*UpdateDepartmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDepartmentRequest)
}

// Partially updates a department. Adding scanning stations or machines is additive and does not remove existing ones.
type UpdateDepartmentEndpoint struct{}

func (e *UpdateDepartmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateDepartmentRequest, *apiresource.Department] {
	return (&apiendpoint.APIEndpoint[*UpdateDepartmentRequest, *apiresource.Department]{
		Title:             "Update Department",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
			return svc.(DepartmentSvc).UpdateDepartment
		},
		ObjectType: constants.ObjectTypeDepartment,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDepartment,
			Fields:     []string{"location", "scanning_stations", "machines"},
		}),
	})
}
