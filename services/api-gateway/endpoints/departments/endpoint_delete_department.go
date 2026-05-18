package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a department.
type DeleteDepartmentRequest struct {
	// Department ID.
	DepartmentID string `path:"id" validate:"required"`
}

// Deletes a department. Fails if the department still has associated scanning stations or machines.
type DeleteDepartmentEndpoint struct{}

func (e *DeleteDepartmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteDepartmentRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteDepartmentRequest, *apiresource.EmptyResource]{
		Title:             "Delete Department",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteDepartmentRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(DepartmentSvc).DeleteDepartment
		},
	})
}
