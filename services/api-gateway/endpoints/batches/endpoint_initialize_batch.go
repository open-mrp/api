package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to initialize a batch at a scanning station.
type InitializeBatchRequest struct {
	// ID of the batch to initialize.
	//
	// The batch must belong to a production run, still be open, and not have been scanned before.
	BatchID string `json:"batch_id" validate:"required"`
	// ID of the scanning station the batch is being scanned at.
	//
	// The station must have a production step that produces the batch's item, since that step is what the batch is attached to.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
	// Station type to scan as, when it differs from the station's own.
	//
	// Scanning as `init_batch` at a station of another type initializes the batch into the one step the station runs for the batch's item; when it runs several, `production_step_id` must pick one. Requires the `update` permission on scanning stations when it differs from the station's type.
	TypeOverride field.Optional[constants.ScanningStationType] `json:"type_override,omitzero"`
	// ID of the production step to initialize the batch into, when the station runs more than one step for the batch's item.
	//
	// Must be one of the steps the station runs that produce the batch's item. When omitted, the batch is initialized into the station's step for the item that has no upstream step.
	ProductionStepID field.Optional[string] `json:"production_step_id,omitzero"`
	// Whether the scan consumes the step's materials and produces its inventory. Defaults to `true`.
	//
	// `false` only marks the batch as scanned, moving no inventory, and requires the `update` permission on scanning stations.
	ConsumeMaterials field.Optional[bool] `json:"consume_materials,omitzero"`
}

var sampleInitializeBatchRequest = &InitializeBatchRequest{
	BatchID:           apiresource.SampleBatchID,
	ScanningStationID: apiresource.SampleScanningStationID,
}

func (*InitializeBatchRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleInitializeBatchRequest)
}

// Marks a production run batch as scanned at a scanning station, starting it through production.
//
// The batch is attached to the production step that produces its item at the station, the step's material consumption and the batch's produced inventory are recorded asynchronously, and the batch is closed automatically if the step is the last one. The batch's production run is started, and the run is closed once all of its batches are scanned or deleted. A scan beyond the account plan's batch limit for the billing period is rejected.
type InitializeBatchEndpoint struct{}

func (e *InitializeBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*InitializeBatchRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*InitializeBatchRequest, *apiresource.Batch]{
		Title:               "Initialize Batch",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/actions/initialize",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionCreate}},
		ObjectType:          constants.ObjectTypeBatch,
		// A batch reports three measures; the units they are counted in are records of their own.
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeBatch,
			Fields:     []string{"quantity.unit", "seconds.unit", "waste.unit"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *InitializeBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).InitializeBatch
		},
	})
}
