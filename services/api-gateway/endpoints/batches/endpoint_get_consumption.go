package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get material consumption data for a scanning station.
type GetScanningStationConsumptionRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
	// Batch IDs to calculate consumption for.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Production step ID to scope the consumption calculation.
	ProductionStepID *string `json:"production_step_id"`
	// Split quantity to factor into the consumption calculation.
	SplitQuantity *SplitQuantityInput `json:"split_quantity"`
}

var sampleGetScanningStationConsumptionRequest = &GetScanningStationConsumptionRequest{
	BatchIDs:         []string{apiresource.SampleBatchID},
	ProductionStepID: func() *string { s := apiresource.SampleProductionStepID; return &s }(),
	SplitQuantity: &SplitQuantityInput{
		ID:      apiresource.SampleBatchID,
		Measure: "10.5",
		UnitID:  apiresource.SampleUnitID,
	},
}

func (*GetScanningStationConsumptionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleGetScanningStationConsumptionRequest)
}

type GetScanningStationConsumptionEndpoint struct{}

func (e *GetScanningStationConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetScanningStationConsumptionRequest, *apiresource.List[apiresource.ScanningConsumption]] {
	return &apiendpoint.APIEndpoint[*GetScanningStationConsumptionRequest, *apiresource.List[apiresource.ScanningConsumption]]{
		Title:             "Get Scanning Station Consumption",
		Description:       "Returns material consumption data for the specified batches at a scanning station, optionally scoped to a production step and split quantity.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}/consumptions",
		Request:           &GetScanningStationConsumptionRequest{},
		Response:          &apiresource.List[apiresource.ScanningConsumption]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetScanningStationConsumptionRequest) (*apiresource.List[apiresource.ScanningConsumption], *apierror.APIError) {
			return svc.(BatchSvc).GetScanningStationConsumption
		},
	}
}
