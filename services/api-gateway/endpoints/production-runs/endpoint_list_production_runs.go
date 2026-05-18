package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list production runs.
type ListProductionRunsRequest struct {
	apiresource.PaginationRequest
	// Filter by status: "open" or "closed".
	Status *string `query:"status"`
	// Filter by item IDs (batches containing these items).
	ItemIDs []string `query:"item_ids"`
	// Filter by machine IDs (batches using these machines).
	MachineIDs []string `query:"machine_ids"`
	// Filter by start date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter by end date (inclusive).
	EndDate *string `query:"end_date"`
}

// TODO: stop returning ProductionRunSummary; return the full ProductionRun apiresource and use proper includes values to control expansion.

// Returns a paginated list of production runs.
type ListProductionRunsEndpoint struct{}

func (e *ListProductionRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionRunsRequest, *apiresource.List[apiresource.ProductionRunSummary]] {
	return (&apiendpoint.APIEndpoint[*ListProductionRunsRequest, *apiresource.List[apiresource.ProductionRunSummary]]{
		Title:             "List Production Runs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs",
		Request:           &ListProductionRunsRequest{},
		Response:          &apiresource.List[apiresource.ProductionRunSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionRunsRequest) (*apiresource.List[apiresource.ProductionRunSummary], *apierror.APIError) {
			return svc.(ProductionRunSvc).ListProductionRuns
		},
	}).WithDocSource(e)
}
