package departmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Filters which departments land in the exported file.
type ExportDepartmentsRequest struct {
	// Free-text search term matched against department names.
	Query *string `json:"q"`
}

var sampleExportDepartmentsRequest = &ExportDepartmentsRequest{}

func (*ExportDepartmentsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportDepartmentsRequest)
}

// Starts an export of every matching department and returns the job that tracks it.
type ExportDepartmentsEndpoint struct{}

func (e *ExportDepartmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportDepartmentsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportDepartmentsRequest, *apiresource.Job]{
		Title:             "Export Departments",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportDepartmentsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(DepartmentSvc).ExportDepartments
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
