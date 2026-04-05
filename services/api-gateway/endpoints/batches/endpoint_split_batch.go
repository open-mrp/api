package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// SplitQuantityInput represents a quantity input for a split operation.
type SplitQuantityInput struct {
	// An optional identifier for this split quantity.
	ID string `json:"id"`
	// The decimal measure value.
	Measure string `json:"measure" validate:"required"`
	// The ID of the unit for this quantity.
	UnitID string `json:"unit_id" validate:"required"`
}

// SplitBatchRequest is the request to split a batch into multiple parts.
type SplitBatchRequest struct {
	// The IDs of the batches to split.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// The ID of the scanning station performing the split.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
	// The ID of the production step for the split.
	ProductionStepID string `json:"production_step_id" validate:"required"`
	// The first split quantity (required).
	Firsts SplitQuantityInput `json:"firsts" validate:"required"`
	// The second split quantity (optional).
	Seconds *SplitQuantityInput `json:"seconds"`
	// The waste quantity (optional).
	Waste *SplitQuantityInput `json:"waste"`
	// Whether to close the original batch after splitting.
	CloseBatch bool `json:"close_batch"`
}

var sampleSplitBatchRequest = &SplitBatchRequest{
	BatchIDs:          []string{apiresource.SampleBatchID},
	ScanningStationID: apiresource.SampleScanningStationID,
	ProductionStepID:  apiresource.SampleProductionStepID,
	Firsts: SplitQuantityInput{
		ID:      apiresource.SampleBatchID,
		Measure: "10.5",
		UnitID:  apiresource.SampleUnitID,
	},
	CloseBatch: false,
}

func (*SplitBatchRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSplitBatchRequest)
}

type SplitBatchEndpoint struct{}

func (e *SplitBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*SplitBatchRequest, *apiresource.Batch] {
	return &apiendpoint.APIEndpoint[*SplitBatchRequest, *apiresource.Batch]{
		Title:             "Split Batch",
		Description:       "Splits one or more batches into multiple parts with specified quantities, optionally tracking waste and closing the originals.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/batches/actions/split",
		Request:           &SplitBatchRequest{},
		Response:          &apiresource.Batch{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SplitBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).SplitBatch
		},
	}
}
