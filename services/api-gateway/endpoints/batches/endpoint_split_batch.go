package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Quantity input for a split operation.
type SplitQuantityInput struct {
	// Client-side identifier for this quantity.
	//
	// Useful for correlating quantities in your own UI; it is not stored and has no effect on the result.
	ID string `json:"id"`
	// Quantity to split off, as a decimal measure expressed in `unit_id`.
	Measure string `json:"measure" validate:"required"`
	// ID of the unit the measure is expressed in.
	UnitID string `json:"unit_id" validate:"required"`
}

// Request to split a quantity off one or more batches into a new batch.
type SplitBatchRequest struct {
	// Batch IDs to split from.
	//
	// Pass a single ID for single-part production steps, or multiple IDs (one per part) for multi-part steps. Each ID is resolved forward through its production flow to the batch that is actually available at the step, so an operator can scan an earlier batch in the chain.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Scanning station ID performing the split.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
	// The production step the new batch is created at.
	ProductionStepID string `json:"production_step_id" validate:"required"`
	// First-quality output quantity for the new batch.
	//
	// At least one of `firsts`, `seconds`, or `waste` must be non-zero.
	Firsts SplitQuantityInput `json:"firsts" validate:"required"`
	// Seconds-quality (B-grade) output quantity recorded on the new batch.
	//
	// Seconds consume input materials but are not added to inventory.
	Seconds field.Optional[SplitQuantityInput] `json:"seconds,omitzero"`
	// Scrap quantity recorded on the new batch.
	//
	// Like seconds, scrap consumes input materials but is not added to inventory.
	Waste field.Optional[SplitQuantityInput] `json:"waste,omitzero"`
	// Whether to close the source batches after splitting.
	//
	// Set this when the operator is done with the source batch even though quantity is left over. When left open, a source batch is still closed automatically once everything split off it (firsts, seconds, and waste together) accounts for its full quantity.
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

// Splits a quantity off one or more batches into a new batch, grading the output as firsts, seconds, and waste.
//
// Unlike a move, the operator states how much was produced rather than letting the step's ratio decide, which is how partial output and quality grading are recorded. A new batch carrying the firsts quantity is created at the production step, with any seconds and waste recorded on it, and the source batches are linked as inputs. Only the firsts quantity is added to inventory; seconds and waste still consume input materials. The step's material consumption runs asynchronously afterwards, and the new batch is closed immediately if the step is the last one in the flow. Returns the newly created batch.
type SplitBatchEndpoint struct{}

func (e *SplitBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*SplitBatchRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*SplitBatchRequest, *apiresource.Batch]{
		Title:               "Split Batch",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/actions/split",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SplitBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).SplitBatch
		},
	})
}
