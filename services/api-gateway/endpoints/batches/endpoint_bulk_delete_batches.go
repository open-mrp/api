package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete multiple batches.
type DeleteManyBatchesRequest struct {
	// Batch IDs to delete.
	BatchIDs []string `json:"batch_ids" validate:"required"`
}

var sampleDeleteManyBatchesRequest = &DeleteManyBatchesRequest{
	BatchIDs: []string{apiresource.SampleBatchID},
}

func (*DeleteManyBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleDeleteManyBatchesRequest)
}

// Deletes multiple batches in one request.
//
// Batch IDs that cannot be found are skipped; the request fails only if none of the batches exist. Deleting a batch also removes its links to the batches feeding into and out of it, breaking the production flow at that point, and detaches it from any machines. After deletion, any production run whose batches are now all scanned or deleted is closed automatically.
type BulkDeleteBatchesEndpoint struct{}

func (e *BulkDeleteBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteManyBatchesRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteManyBatchesRequest, *apiresource.EmptyResource]{
		Title:               "Bulk Delete Batches",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/actions/bulk-delete",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteManyBatchesRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(BatchSvc).DeleteManyBatches
		},
	})
}
