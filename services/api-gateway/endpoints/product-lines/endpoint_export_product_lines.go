package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
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

func (e *ExportProductLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportProductLinesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportProductLinesRequest, *apiresource.Job]{
		Title:             "Export Product Lines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-lines/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportProductLinesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ProductLineSvc).ExportProductLines
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
