package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to merge multiple batches into one.
type MergeBatchesRequest struct {
	// Batch IDs to merge.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Scanning station ID performing the merge.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
	// Production step ID for the merged batch.
	ProductionStepID string `json:"production_step_id" validate:"required"`
}

var sampleMergeBatchesRequest = &MergeBatchesRequest{
	BatchIDs:          []string{apiresource.SampleBatchID},
	ScanningStationID: apiresource.SampleScanningStationID,
	ProductionStepID:  apiresource.SampleProductionStepID,
}

func (*MergeBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMergeBatchesRequest)
}

// Merges multiple batches into one at the specified production step and scanning station.
type MergeBatchesEndpoint struct{}

func (e *MergeBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*MergeBatchesRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*MergeBatchesRequest, *apiresource.Batch]{
		Title:             "Merge Batches",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/actions/merge",
		Request:           &MergeBatchesRequest{},
		Response:          &apiresource.Batch{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *MergeBatchesRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).MergeBatches
		},
	}).WithDocSource(e)
}
