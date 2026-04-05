package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteManyBatchesRequest is the request to delete multiple batches at once.
type DeleteManyBatchesRequest struct {
	// The IDs of the batches to delete.
	BatchIDs []string `json:"batch_ids" validate:"required"`
}

var sampleDeleteManyBatchesRequest = &DeleteManyBatchesRequest{
	BatchIDs: []string{apiresource.SampleBatchID},
}

func (*DeleteManyBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleDeleteManyBatchesRequest)
}

type BulkDeleteBatchesEndpoint struct{}

func (e *BulkDeleteBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteManyBatchesRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteManyBatchesRequest, *apiresource.EmptyResource]{
		Title:             "Bulk Delete Batches",
		Description:       "Deletes multiple batches in a single operation.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/batches/actions/bulk-delete",
		Request:           &DeleteManyBatchesRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteManyBatchesRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(BatchSvc).DeleteManyBatches
		},
	}
}
