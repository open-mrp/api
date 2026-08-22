package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Filters which production runs land in the exported file.
type ExportProductionRunsRequest struct {
	// Free-text search term matched against production run numbers.
	Query *string `json:"q"`
}

var sampleExportProductionRunsRequest = &ExportProductionRunsRequest{}

func (*ExportProductionRunsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportProductionRunsRequest)
}

// Starts an export of every matching production run and returns the job that tracks it; the file
// lists a run's batches one per row.
type ExportProductionRunsEndpoint struct{}

func (e *ExportProductionRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportProductionRunsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportProductionRunsRequest, *apiresource.Job]{
		Title:             "Export Production Runs",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportProductionRunsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ProductionRunSvc).ExportProductionRuns
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
