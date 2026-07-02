package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve the production flow graph for a batch.
type GetBatchFlowRequest struct {
	// Batch ID.
	BatchID string `path:"id" validate:"required"`
}

// Returns the full production flow graph containing a batch.
//
// The flow is every batch connected to the given batch through input/output relationships, in both directions, returned as nodes with their input and output edges.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetBatchFlowRequest) (*apiresource.List[apiresource.BatchFlowNode], *apierror.APIError) {
			return svc.(BatchSvc).GetBatchFlow
		},
	})
}
