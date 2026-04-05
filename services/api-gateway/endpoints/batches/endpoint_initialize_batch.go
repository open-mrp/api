package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// InitializeBatchRequest is the request to initialize a new batch at a scanning station.
type InitializeBatchRequest struct {
	// The ID of the batch to initialize.
	BatchID string `json:"batch_id" validate:"required"`
	// The ID of the scanning station where the batch is being initialized.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
}

var sampleInitializeBatchRequest = &InitializeBatchRequest{
	BatchID:           apiresource.SampleBatchID,
	ScanningStationID: apiresource.SampleScanningStationID,
}

func (*InitializeBatchRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleInitializeBatchRequest)
}

type InitializeBatchEndpoint struct{}

func (e *InitializeBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*InitializeBatchRequest, *apiresource.Batch] {
	return &apiendpoint.APIEndpoint[*InitializeBatchRequest, *apiresource.Batch]{
		Title:             "Initialize Batch",
		Description:       "Initializes a batch at the specified scanning station.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/batches/actions/initialize",
		Request:           &InitializeBatchRequest{},
		Response:          &apiresource.Batch{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *InitializeBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).InitializeBatch
		},
	}
}
