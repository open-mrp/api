package departmentep

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

// Request to partially update a department.
type UpdateDepartmentRequest struct {
	// Department ID.
	DepartmentID string `path:"id" validate:"required"`
	// Display name of the department.
	//
	// Must be unique within your account; maximum 255 characters.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Free-form notes about the department.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// ID of the location where this department operates.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// IDs of scanning stations to assign to this department.
	//
	// Assignment is additive: listed stations are moved into this department and stations already in the department are unaffected.
	ScanningStationIDs []string `json:"scanning_station_ids,omitzero"`
	// IDs of machines to assign to this department.
	//
	// Assignment is additive: listed machines are moved into this department and machines already in the department are unaffected.
	MachineIDs []string `json:"machine_ids,omitzero"`
}

var sampleUpdateDepartmentName = "Production"
var sampleUpdateDepartmentRequest = &UpdateDepartmentRequest{
	Name: field.Some(sampleUpdateDepartmentName),
}

func (*UpdateDepartmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDepartmentRequest)
}

// Partially updates a department.
//
// Only the fields provided in the request are changed. Assigning scanning stations or machines is additive and does not remove existing ones. Returns a conflict error if the new name is already in use by another department.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeDepartment,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDepartment,
			Fields:     []string{"location", "scanning_stations", "machines"},
		}),
	})
}
