package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a batch.
type DeleteBatchRequest struct {
	// Batch ID.
	BatchID string `path:"id" validate:"required"`
}

// Deletes a batch by ID and returns the deleted batch.
//
// Deleting a batch also removes its links to the batches feeding into and out of it, breaking the production flow at that point, and detaches it from any machines. After deletion, the batch's production run is closed automatically once all of its batches are scanned or deleted. Deleting the same batch twice reports that it has already been deleted.
type DeleteBatchEndpoint struct{}

func (e *DeleteBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteBatchRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*DeleteBatchRequest, *apiresource.Batch]{
		Title:               "Delete Batch",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).DeleteBatch
		},
	})
}
