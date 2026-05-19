package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a department.
type UpdateDepartmentRequest struct {
	// Department ID.
	DepartmentID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Notes about the department.
	Notes *string `json:"notes,omitempty"`
	// Storage location ID.
	LocationID *string `json:"location_id,omitempty" validate:"omitempty"`
	// Scanning station IDs to connect (additive).
	ScanningStationIDs []string `json:"scanning_station_ids,omitempty"`
	// Machine IDs to connect (additive).
	MachineIDs []string `json:"machine_ids,omitempty"`
}

var sampleUpdateDepartmentName = "Production"
var sampleUpdateDepartmentRequest = &UpdateDepartmentRequest{
	Name: &sampleUpdateDepartmentName,
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
	})
}
