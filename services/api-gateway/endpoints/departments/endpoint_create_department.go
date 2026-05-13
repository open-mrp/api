package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a department.
type CreateDepartmentRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Notes about the department.
	Notes *string `json:"notes,omitempty"`
	// Storage location ID.
	LocationID *string `json:"location_id,omitempty" validate:"omitempty"`
	// Scanning station IDs to connect.
	ScanningStationIDs []string `json:"scanning_station_ids,omitempty"`
	// Machine IDs to connect.
	MachineIDs []string `json:"machine_ids,omitempty"`
}

var sampleCreateDepartmentRequest = &CreateDepartmentRequest{
	Name:               apiresource.SampleDepartmentName,
	ScanningStationIDs: []string{apiresource.SampleScanningStationID},
	MachineIDs:         []string{apiresource.SampleMachineID},
}

func (*CreateDepartmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateDepartmentRequest)
}

type CreateDepartmentEndpoint struct{}

func (e *CreateDepartmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateDepartmentRequest, *apiresource.Department] {
	return &apiendpoint.APIEndpoint[*CreateDepartmentRequest, *apiresource.Department]{
		Title:             "Create Department",
		Description:       "Creates a department.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments",
		Request:           &CreateDepartmentRequest{},
		Response:          &apiresource.Department{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
			return svc.(DepartmentSvc).CreateDepartment
		},
		LocationFunc: func(resp *apiresource.Department) string {
			return "/v1/operations/departments/" + resp.ID
		},
	}
}
