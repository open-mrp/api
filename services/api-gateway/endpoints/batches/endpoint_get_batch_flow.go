package batchep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
		Title:             "Get Batch Flow",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/{id}/flow",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeBatchFlowNode,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetBatchFlowRequest) (*apiresource.List[apiresource.BatchFlowNode], *apierror.APIError) {
			return svc.(BatchSvc).GetBatchFlow
		},
	})
}
