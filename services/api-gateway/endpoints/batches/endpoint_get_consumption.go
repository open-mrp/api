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
	// Batch IDs to calculate consumption for.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Production step ID to scope the consumption calculation.
	//
	// Required when the scanning station is a move, split, or merge station. Ignored for initialize stations, where the step is derived from the station and the batch's item.
	ProductionStepID field.Optional[string] `json:"production_step_id,omitzero"`
	// Proposed split quantity to factor into the consumption calculation.
	//
	// Required when the scanning station is a split station; for a single-batch split, material demand is scaled to this quantity instead of the full batch quantity.
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
// Demand is calculated from the production step's configured consumptions, scaled to the batch quantities (or the proposed split quantity). How the step is determined depends on the station's type: initialize stations derive it from the station and the batch's item, while move, split, and merge stations use `production_step_id`.
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
