package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list batches for a production run.
type ListBatchesByProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
}

// Returns a paginated list of the batches that make up a production run, most recently created first.
//
// The result is not limited to the batches recorded directly against the run. Starting from those batches, the batch flow is followed downstream to the batches they feed and upstream to the batches that feed them while that branch is still open, so the whole in-progress flow around the run is returned.
//
// The `q` search term matches a batch ID, item SKU, scanning station name, department name, production step name, run number, lot number, or machine name.
type ListBatchesByProductionRunEndpoint struct{}

func (e *ListBatchesByProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListBatchesByProductionRunRequest, *apiresource.List[apiresource.Batch]] {
	return (&apiendpoint.APIEndpoint[*ListBatchesByProductionRunRequest, *apiresource.List[apiresource.Batch]]{
		Title:             "List Batches by Production Run",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}/batches",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListBatchesByProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
			return svc.(ProductionRunSvc).ListBatchesByProductionRun
		},
	})
}
