package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a department.
type RetrieveDepartmentRequest struct {
	// ID of the department to retrieve.
	DepartmentID string `path:"id" validate:"required"`
}

// Returns a department by ID.
type RetrieveDepartmentEndpoint struct{}

func (e *RetrieveDepartmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveDepartmentRequest, *apiresource.Department] {
	return (&apiendpoint.APIEndpoint[*RetrieveDepartmentRequest, *apiresource.Department]{
		Title:             "Retrieve Department",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
			return svc.(DepartmentSvc).GetDepartment
		},
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeDepartment,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDepartment,
			Fields:     []string{"location", "scanning_stations", "machines"},
		}),
	})
}
