package productionrunep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list batches for a production run.
type ListBatchesByProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
	// Which batches to return.
	//
	// `flow` (the default) follows the batch flow around the run's batches as described above. `run` returns only the batches created under the run itself, as they were planned on it.
	Scope *constants.ProductionRunBatchScope `query:"scope"`
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of the batches that make up a production run, most recently created first.
//
// By default the result is not limited to the batches recorded directly against the run. Starting from those batches, the batch flow is followed downstream to the batches they feed and upstream to the batches that feed them while that branch is still open, so the whole in-progress flow around the run is returned. Pass `scope=run` for only the run's own batches.
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
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionRead},
		}},
		ObjectType: constants.ObjectTypeBatch,
		// A batch reports three measures; the units they are counted in are records of their own.
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeBatch,
			Fields:     []string{"quantity.unit", "seconds.unit", "waste.unit"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListBatchesByProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
			return svc.(ProductionRunSvc).ListBatchesByProductionRun
		},
	})
}
