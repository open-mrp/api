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

// Request to get material consumption data for a scanning station.
type GetScanningStationConsumptionRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
	// Batch IDs the scanning operation would be performed on.
	//
	// At an `init_batch` station only the first ID is used, since demand comes from that batch's own item and quantity. At the other station types each ID is resolved forward through its production flow to the batch that is actually available at the step, and the demand of all of them is added together.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Production step ID to scope the consumption calculation.
	//
	// Required for `move_batch`, `split_batch`, and `merge_batch` stations. Ignored for `init_batch` stations, where the step is derived from the station and the batch's item.
	ProductionStepID field.Optional[string] `json:"production_step_id,omitzero"`
	// Proposed split quantity to factor into the consumption calculation.
	//
	// Required for `split_batch` stations. It is applied only when splitting a single batch at a single-part step, where material demand is scaled to this quantity instead of the batch's full expected output; splits covering several batches or several parts ignore it.
	SplitQuantity field.Optional[SplitQuantityInput] `json:"split_quantity,omitzero"`
}

var sampleGetScanningStationConsumptionRequest = &GetScanningStationConsumptionRequest{
	BatchIDs:         []string{apiresource.SampleBatchID},
	ProductionStepID: field.Some(apiresource.SampleProductionStepID),
	SplitQuantity: field.Some(SplitQuantityInput{
		ID:      apiresource.SampleBatchID,
		Measure: "10.5",
		UnitID:  apiresource.SampleUnitID,
	}),
}

func (*GetScanningStationConsumptionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleGetScanningStationConsumptionRequest)
}

// Returns the material demand and current inventory for the operation a scanning station would perform on the given batches.
//
// Use this to preview what a scan will draw from stock, and to compare that against what is on hand, before committing the operation. Demand is each of the step's configured material quantities scaled by how much output the operation produces relative to the step's standard run size, so it grows with the batch quantities (or the proposed split quantity). How the step is determined depends on the station's type: `init_batch` stations derive it from the station and the batch's item, while `move_batch`, `split_batch`, and `merge_batch` stations use `production_step_id`. Nothing is consumed by this call.
type GetScanningStationConsumptionEndpoint struct{}

func (e *GetScanningStationConsumptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetScanningStationConsumptionRequest, *apiresource.List[apiresource.ScanningConsumption]] {
	return (&apiendpoint.APIEndpoint[*GetScanningStationConsumptionRequest, *apiresource.List[apiresource.ScanningConsumption]]{
		Title:               "Get Scanning Station Consumption",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/scanning-stations/{id}/consumptions",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetScanningStationConsumptionRequest) (*apiresource.List[apiresource.ScanningConsumption], *apierror.APIError) {
			return svc.(BatchSvc).GetScanningStationConsumption
		},
	})
}
