package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListDepartmentsRequest is the request to list departments with optional filters.
type ListDepartmentsRequest struct {
	apiresource.PaginationRequest
}

type ListDepartmentsEndpoint struct{}

func (e *ListDepartmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListDepartmentsRequest, *apiresource.List[apiresource.Department]] {
	return &apiendpoint.APIEndpoint[*ListDepartmentsRequest, *apiresource.List[apiresource.Department]]{
		Title:             "List Departments",
		Description:       "Returns a paginated list of departments for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/departments",
		Request:           &ListDepartmentsRequest{},
		Response:          &apiresource.List[apiresource.Department]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDepartmentsRequest) (*apiresource.List[apiresource.Department], *apierror.APIError) {
			return svc.(DepartmentSvc).ListDepartments
		},
	}
}
