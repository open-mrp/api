package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// MergeBatchesRequest is the request to merge multiple batches into one.
type MergeBatchesRequest struct {
	// The IDs of the batches to merge.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// The ID of the scanning station performing the merge.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
	// The ID of the production step for the merged batch.
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

type MergeBatchesEndpoint struct{}

func (e *MergeBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*MergeBatchesRequest, *apiresource.Batch] {
	return &apiendpoint.APIEndpoint[*MergeBatchesRequest, *apiresource.Batch]{
		Title:             "Merge Batches",
		Description:       "Merges multiple batches into a single batch at the specified production step and scanning station.",
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
	}
}
