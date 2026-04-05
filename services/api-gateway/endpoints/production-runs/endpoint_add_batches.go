package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AddBatchInputRequest represents a single batch to add to a production run.
type AddBatchInputRequest struct {
	// The item ID for the batch.
	ItemID string `json:"item_id" validate:"required"`
	// The quantity value as a decimal string.
	QuantityValue string `json:"quantity_value" validate:"required"`
	// The unit ID for the quantity.
	QuantityUnitID string `json:"quantity_unit_id" validate:"required"`
	// The seconds value as a decimal string.
	SecondsValue *string `json:"seconds_value"`
	// The unit ID for seconds.
	SecondsUnitID *string `json:"seconds_unit_id"`
	// The waste value as a decimal string.
	WasteValue *string `json:"waste_value"`
	// The unit ID for waste.
	WasteUnitID *string `json:"waste_unit_id"`
	// The production step ID.
	ProductionStepID *string `json:"production_step_id"`
	// The scanning station ID.
	ScanningStationID *string `json:"scanning_station_id"`
}

// AddBatchesToProductionRunRequest is the request to add batches to a production run.
type AddBatchesToProductionRunRequest struct {
	// The ID of the production run.
	ProductionRunID string `path:"id" validate:"required"`
	// The batches to add.
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

type AddBatchesToProductionRunEndpoint struct{}

func (e *AddBatchesToProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddBatchesToProductionRunRequest, *apiresource.List[apiresource.Batch]] {
	return &apiendpoint.APIEndpoint[*AddBatchesToProductionRunRequest, *apiresource.List[apiresource.Batch]]{
		Title:             "Add Batches to Production Run",
		Description:       "Adds batches to a production run. Fails if the run is already completed.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/production-runs/{id}/batches",
		Request:           &AddBatchesToProductionRunRequest{},
		Response:          &apiresource.List[apiresource.Batch]{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddBatchesToProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
			return svc.(ProductionRunSvc).AddBatchesToProductionRun
		},
	}
}
