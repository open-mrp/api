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
	apiexample "github.com/open-mrp/apikit/example"
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

func (e *ExportDepartmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportDepartmentsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportDepartmentsRequest, *apiresource.AsyncJob]{
		Title:             "Export Departments",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/departments/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDepartments, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportDepartmentsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(DepartmentSvc).ExportDepartments
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
