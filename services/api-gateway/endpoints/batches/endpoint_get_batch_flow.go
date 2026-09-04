package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve the production flow graph for a batch.
type GetBatchFlowRequest struct {
	// Batch ID.
	BatchID string `path:"id" validate:"required"`
}

// Returns the full production flow graph containing a batch.
//
// The flow is every batch connected to the given batch through input/output relationships, in both directions, including the batch itself. Nodes come back in no particular order; rebuild the graph from each node's input and output edges rather than from their position in the list.
type GetBatchFlowEndpoint struct{}

func (e *GetBatchFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetBatchFlowRequest, *apiresource.List[apiresource.BatchFlowNode]] {
	return (&apiendpoint.APIEndpoint[*GetBatchFlowRequest, *apiresource.List[apiresource.BatchFlowNode]]{
		Title:               "Get Batch Flow",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/{id}/flow",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeBatchFlowNode,
		// Each node carries a whole batch, so its measures are reached through that batch.
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeBatchFlowNode,
			Fields:     []string{"batch.quantity.unit", "batch.seconds.unit", "batch.waste.unit"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetBatchFlowRequest) (*apiresource.List[apiresource.BatchFlowNode], *apierror.APIError) {
			return svc.(BatchSvc).GetBatchFlow
		},
	})
}
