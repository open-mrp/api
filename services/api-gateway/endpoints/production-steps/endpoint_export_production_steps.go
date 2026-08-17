package productionstepep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Filters which production steps land in the exported file.
type ExportProductionStepsRequest struct {
	// Free-text search term matched against production step names.
	Query *string `json:"q"`
}

var sampleExportProductionStepsRequest = &ExportProductionStepsRequest{}

func (*ExportProductionStepsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportProductionStepsRequest)
}

// Starts an export of every matching production step and returns the job that tracks it; the file
// lists a step's consumptions one per row.
type ExportProductionStepsEndpoint struct{}

func (e *ExportProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportProductionStepsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*ExportProductionStepsRequest, *apiresource.Job]{
		Title:             "Export Production Steps",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportProductionStepsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ProductionStepSvc).ExportProductionSteps
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
