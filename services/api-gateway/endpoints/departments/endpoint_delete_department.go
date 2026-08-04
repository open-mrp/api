package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a department.
type DeleteDepartmentRequest struct {
	// ID of the department to delete.
	DepartmentID string `path:"id" validate:"required"`
}

// Deletes a department.
//
// Scanning stations and machines assigned to the department are not deleted, but they keep pointing at it, and a machine whose department is gone can no longer be read, updated, or deleted through the machines endpoints. Reassign both to another department before deleting this one. Deleting a department that was already deleted returns an already-deleted error rather than a not-found error.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionDelete},
		},
	})
}
