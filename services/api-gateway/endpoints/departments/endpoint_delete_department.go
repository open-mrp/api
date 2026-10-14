package departmentep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteDepartmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteDepartmentRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteDepartmentRequest, *object.Deleted]{
		Title:             "Delete Department",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeDepartment,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteDepartmentRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(DepartmentSvc).DeleteDepartment, constants.ObjectTypeDepartment, "id")
		},
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDepartments, Action: types.ActionDelete},
		}},
	})
}
