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
		ObjectType:          constants.ObjectTypeBatch,
		// A batch reports three measures; the units they are counted in are records of their own.
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeBatch,
			Fields:     []string{"quantity.unit", "seconds.unit", "waste.unit"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).DeleteBatch
		},
	})
}
