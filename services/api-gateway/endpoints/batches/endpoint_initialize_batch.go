package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to initialize a batch at a scanning station.
type InitializeBatchRequest struct {
	// Batch ID.
	BatchID string `json:"batch_id" validate:"required"`
	// Scanning station ID.
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
		ContentType:       "application/json",
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
