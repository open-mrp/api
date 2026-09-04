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
// The batch is attached to the production step that produces its item at the station, the step's material consumption is executed asynchronously, and the batch is closed automatically if the step is the last one. The batch's production run is started, and the run is closed once all of its batches are scanned or deleted.
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
