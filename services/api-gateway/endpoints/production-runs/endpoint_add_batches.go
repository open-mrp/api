package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Batch to add to a production run.
type AddBatchInputRequest struct {
	// Item ID.
	ItemID string `json:"item_id" validate:"required"`
	// Quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// Quantity unit ID.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
	// Seconds value as a decimal string.
	SecondsValue *string `json:"seconds_value"`
	// Seconds unit ID.
	SecondsUnitID *string `json:"seconds_unit_id"`
	// Waste value as a decimal string.
	WasteValue *string `json:"waste_value"`
	// Waste unit ID.
	WasteUnitID *string `json:"waste_unit_id"`
	// Production step ID.
	ProductionStepID *string `json:"production_step_id"`
	// Scanning station ID.
	ScanningStationID *string `json:"scanning_station_id"`
}

// Request to add batches to a production run.
type AddBatchesToProductionRunRequest struct {
	// Production run ID.
	ProductionRunID string `path:"id" validate:"required"`
	// Batches to add.
	Batches []AddBatchInputRequest `json:"batches" validate:"required,min=1"`
}

var sampleAddBatchesProductionStepID = apiresource.SampleProductionStepID
var sampleAddBatchesToProductionRunRequest = &AddBatchesToProductionRunRequest{
	Batches: []AddBatchInputRequest{
		{
			ItemID:           apiresource.SampleItemID,
			QuantityValue:    "100",
			QuantityUnitID:   apiresource.SampleUnitID,
			ProductionStepID: &sampleAddBatchesProductionStepID,
		},
	},
}

func (*AddBatchesToProductionRunRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddBatchesToProductionRunRequest)
}

// Adds batches to a production run. Fails if the run is completed.
type AddBatchesToProductionRunEndpoint struct{}

func (e *AddBatchesToProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddBatchesToProductionRunRequest, *apiresource.List[apiresource.Batch]] {
	return (&apiendpoint.APIEndpoint[*AddBatchesToProductionRunRequest, *apiresource.List[apiresource.Batch]]{
		Title:             "Add Batches to Production Run",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs/{id}/batches",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddBatchesToProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
			return svc.(ProductionRunSvc).AddBatchesToProductionRun
		},
	})
}
