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

// Request to create a department.
type CreateDepartmentRequest struct {
	// Display name of the department.
	//
	// Must be unique within your account; maximum 255 characters.
	Name string `json:"name" validate:"required,max=255"`
	// Free-form notes about the department.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// ID of the location where this department operates.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// IDs of scanning stations to assign to this department.
	//
	// A scanning station belongs to one department at a time, so listed stations are moved out of their current department.
	ScanningStationIDs []string `json:"scanning_station_ids,omitzero"`
	// IDs of machines to assign to this department.
	//
	// A machine belongs to one department at a time, so listed machines are moved out of their current department.
	MachineIDs []string `json:"machine_ids,omitzero"`
}

var sampleCreateDepartmentRequest = &CreateDepartmentRequest{
	Name:               apiresource.SampleDepartmentName,
	ScanningStationIDs: []string{apiresource.SampleScanningStationID},
	MachineIDs:         []string{apiresource.SampleMachineID},
}

func (*CreateDepartmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateDepartmentRequest)
}

// Creates a department, optionally assigning scanning stations and machines to it.
//
// Returns a conflict error if a department with the same name already exists.
type CreateDepartmentEndpoint struct{}

func (e *CreateDepartmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateDepartmentRequest, *apiresource.Department] {
	return (&apiendpoint.APIEndpoint[*CreateDepartmentRequest, *apiresource.Department]{
		Title:             "Create Department",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
			return svc.(DepartmentSvc).CreateDepartment
		},
		LocationFunc: func(resp *apiresource.Department) string {
			return "/v1/operations/departments/" + resp.ID
		},
		ObjectType: constants.ObjectTypeDepartment,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDepartment,
			Fields:     []string{"location", "scanning_stations", "machines"},
		}),
	})
}
