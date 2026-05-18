package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a batch.
type DeleteBatchRequest struct {
	// Batch ID.
	BatchID string `path:"id" validate:"required"`
}

// Deletes a batch by ID.
type DeleteBatchEndpoint struct{}

func (e *DeleteBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteBatchRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*DeleteBatchRequest, *apiresource.Batch]{
		Title:             "Delete Batch",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).DeleteBatch
		},
	})
}
