package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateDepartmentRequest is the request to partially update a department.
type UpdateDepartmentRequest struct {
	// The ID of the department to update.
	DepartmentID string `path:"id" validate:"required"`
	// The display name of the department.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Optional notes about the department.
	Notes *string `json:"notes,omitempty" nullable:"true"`
	// The ID of the storage location to associate with this department.
	LocationID *string `json:"location_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// IDs of scanning stations to connect to this department (additive).
	ScanningStationIDs []string `json:"scanning_station_ids,omitempty"`
	// IDs of machines to connect to this department (additive).
	MachineIDs []string `json:"machine_ids,omitempty"`
}

var sampleUpdateDepartmentName = "Production"
var sampleUpdateDepartmentRequest = &UpdateDepartmentRequest{
	Name: &sampleUpdateDepartmentName,
}

func (*UpdateDepartmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDepartmentRequest)
}

type UpdateDepartmentEndpoint struct{}

func (e *UpdateDepartmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateDepartmentRequest, *apiresource.Department] {
	return &apiendpoint.APIEndpoint[*UpdateDepartmentRequest, *apiresource.Department]{
		Title:             "Update Department",
		Description:       "Partially updates a department. Adding scanning stations or machines is additive and does not remove existing ones.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/{id}",
		Request:           &UpdateDepartmentRequest{},
		Response:          &apiresource.Department{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
			return svc.(DepartmentSvc).UpdateDepartment
		},
	}
}
