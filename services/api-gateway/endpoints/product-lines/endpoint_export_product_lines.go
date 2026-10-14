package productlineep

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

// Filters which product lines land in the exported file.
type ExportProductLinesRequest struct {
	// Free-text search term matched against product line names.
	Query *string `json:"q"`
}

var sampleExportProductLinesRequest = &ExportProductLinesRequest{}

func (*ExportProductLinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportProductLinesRequest)
}

// Starts an export of every matching product line and returns the job that tracks it; system
type ExportProductLinesEndpoint struct{}

func (e *ExportProductLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportProductLinesRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportProductLinesRequest, *apiresource.AsyncJob]{
		Title:             "Export Product Lines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-lines/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportProductLinesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(ProductLineSvc).ExportProductLines
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
