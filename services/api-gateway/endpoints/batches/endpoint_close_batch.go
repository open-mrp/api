package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to close a batch.
type CloseBatchRequest struct {
	// Batch ID.
	BatchID string `json:"batch_id" validate:"required"`
}

var sampleCloseBatchRequest = &CloseBatchRequest{
	BatchID: apiresource.SampleBatchID,
}

func (*CloseBatchRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCloseBatchRequest)
}

// Closes a batch, marking it as completed.
type CloseBatchEndpoint struct{}

func (e *CloseBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*CloseBatchRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*CloseBatchRequest, *apiresource.Batch]{
		Title:             "Close Batch",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/actions/close",
		Request:           &CloseBatchRequest{},
		Response:          &apiresource.Batch{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CloseBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).CloseBatch
		},
	}).WithDocSource(e)
}
