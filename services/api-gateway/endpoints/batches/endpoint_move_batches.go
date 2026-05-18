package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to move batches to a production step.
type MoveBatchesRequest struct {
	// Batch IDs to move.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Target production step ID.
	ProductionStepID string `json:"production_step_id" validate:"required"`
	// Scanning station ID performing the move.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
}

var sampleMoveBatchesRequest = &MoveBatchesRequest{
	BatchIDs:          []string{apiresource.SampleBatchID},
	ProductionStepID:  apiresource.SampleProductionStepID,
	ScanningStationID: apiresource.SampleScanningStationID,
}

func (*MoveBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMoveBatchesRequest)
}

// Moves batches to a production step at the specified scanning station.
type MoveBatchesEndpoint struct{}

func (e *MoveBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*MoveBatchesRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*MoveBatchesRequest, *apiresource.Batch]{
		Title:             "Move Batches",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/actions/move",
		Request:           &MoveBatchesRequest{},
		Response:          &apiresource.Batch{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *MoveBatchesRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).MoveBatches
		},
	}).WithDocSource(e)
}
