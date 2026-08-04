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

// Request to list departments.
type ListDepartmentsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of departments in your account, most recently created first.
//
// The `q` search term matches the department name.
type ListDepartmentsEndpoint struct{}

func (e *ListDepartmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListDepartmentsRequest, *apiresource.List[apiresource.Department]] {
	return (&apiendpoint.APIEndpoint[*ListDepartmentsRequest, *apiresource.List[apiresource.Department]]{
		Title:             "List Departments",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDepartmentsRequest) (*apiresource.List[apiresource.Department], *apierror.APIError) {
			return svc.(DepartmentSvc).ListDepartments
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
