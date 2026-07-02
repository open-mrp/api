package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Quantity input for a split operation.
type SplitQuantityInput struct {
	// Identifier for this split quantity.
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
	// Pass a single ID for single-part production steps, or multiple IDs (one per part) for multi-part steps.
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
	Waste field.Optional[SplitQuantityInput] `json:"waste,omitzero"`
	// Whether to close the source batches after splitting.
	//
	// When the source batches are left open, each is still closed automatically once its quantity is fully used by splits.
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
// A new batch carrying the firsts quantity is created at the production step, with any seconds and waste recorded on it; the source batches are linked as inputs. Returns the newly created batch.
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
