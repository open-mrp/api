package workcenterep

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

// Filters which work centers land in the exported file.
type ExportWorkCentersRequest struct {
	// Free-text search term matched against work center names.
	Query *string `json:"q"`
}

var sampleExportWorkCentersRequest = &ExportWorkCentersRequest{}

func (*ExportWorkCentersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportWorkCentersRequest)
}

// Starts an export of every matching work center and returns the job that tracks it.
type ExportWorkCentersEndpoint struct{}

func (e *ExportWorkCentersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportWorkCentersRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportWorkCentersRequest, *apiresource.AsyncJob]{
		Title:             "Export Work Centers",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/work-centers/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDepartments, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportWorkCentersRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(WorkCenterSvc).ExportWorkCenters
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
