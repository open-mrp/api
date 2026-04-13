package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a department.
type GetDepartmentRequest struct {
	// Department ID.
	DepartmentID string `path:"id" validate:"required"`
}

type GetDepartmentEndpoint struct{}

func (e *GetDepartmentEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetDepartmentRequest, *apiresource.Department] {
	return &apiendpoint.APIEndpoint[*GetDepartmentRequest, *apiresource.Department]{
		Title:             "Get Department",
		Description:       "Returns a department by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/{id}",
		Request:           &GetDepartmentRequest{},
		Response:          &apiresource.Department{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
			return svc.(DepartmentSvc).GetDepartment
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDepartment,
			Fields:     []string{"location", "scanning_stations", "machines"},
		}),
	}
}
