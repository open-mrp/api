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

// Request to move batches to a production step.
type MoveBatchesRequest struct {
	// Batch IDs to move.
	//
	// Pass a single ID to advance one batch, or multiple IDs (one per part) when the target step combines multiple parts.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Target production step ID.
	ProductionStepID string `json:"production_step_id" validate:"required"`
	// Scanning station ID performing the move.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
}

var sampleMoveBatchesRequest = &MoveBatchesRequest{
	BatchIDs:          []string{apiresource.SampleBatchID},
	ProductionStepID:  apiresource.SampleProductionStepID,
	ScanningStationID: apiresource.SampleScanningStationID,
}

func (*MoveBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMoveBatchesRequest)
}

// Advances batches to a production step by creating a new batch at that step.
//
// A new batch is created with its item and quantity calculated from the target step's configuration, the source batches are linked as inputs and closed, and the step's material consumption is executed asynchronously. Returns the newly created batch.
type MoveBatchesEndpoint struct{}

func (e *MoveBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*MoveBatchesRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*MoveBatchesRequest, *apiresource.Batch]{
		Title:               "Move Batches",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/actions/move",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MoveBatchesRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).MoveBatches
		},
	})
}
