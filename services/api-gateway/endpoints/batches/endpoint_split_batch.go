package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Quantity input for a split operation.
type SplitQuantityInput struct {
	// Identifier for this split quantity.
	ID string `json:"id"`
	// Decimal measure value.
	Measure string `json:"measure" validate:"required"`
	// Unit ID.
	UnitID string `json:"unit_id" validate:"required"`
}

// Request to split batches into multiple parts.
type SplitBatchRequest struct {
	// Batch IDs to split.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Scanning station ID performing the split.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
	// Production step ID for the split.
	ProductionStepID string `json:"production_step_id" validate:"required"`
	// First split quantity.
	Firsts SplitQuantityInput `json:"firsts" validate:"required"`
	// Second split quantity.
	Seconds *SplitQuantityInput `json:"seconds"`
	// Waste quantity.
	Waste *SplitQuantityInput `json:"waste"`
	// Whether to close the original batches after splitting.
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
		ContentType:       "application/json",
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
