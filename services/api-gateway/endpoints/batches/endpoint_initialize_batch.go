package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to initialize a batch at a scanning station.
type InitializeBatchRequest struct {
	// ID of the batch to initialize; the batch must be open and not yet scanned.
	BatchID string `json:"batch_id" validate:"required"`
	// Scanning station ID.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *InitializeBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).InitializeBatch
		},
	})
}
