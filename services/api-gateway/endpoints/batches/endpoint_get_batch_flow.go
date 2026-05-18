package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve the production flow graph for a batch.
type GetBatchFlowRequest struct {
	// Batch ID.
	BatchID string `path:"id" validate:"required"`
}

// Returns the production flow graph for a batch, including all input and output batch relationships.
type GetBatchFlowEndpoint struct{}

func (e *GetBatchFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetBatchFlowRequest, *apiresource.List[apiresource.BatchFlowNode]] {
	return (&apiendpoint.APIEndpoint[*GetBatchFlowRequest, *apiresource.List[apiresource.BatchFlowNode]]{
		Title:             "Get Batch Flow",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/{id}/flow",
		Request:           &GetBatchFlowRequest{},
		Response:          &apiresource.List[apiresource.BatchFlowNode]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetBatchFlowRequest) (*apiresource.List[apiresource.BatchFlowNode], *apierror.APIError) {
			return svc.(BatchSvc).GetBatchFlow
		},
	}).WithDocSource(e)
}
