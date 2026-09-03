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

// Request to move batches to a production step.
type MoveBatchesRequest struct {
	// Batch IDs to move.
	//
	// Pass a single ID to advance one batch, or multiple IDs (one per part) when the target step combines multiple parts. Each ID is resolved forward through its production flow to the batch that is actually available at the step, so an operator can scan an earlier batch in the chain.
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
// The new batch carries the item the target step produces, and its quantity is scaled from the source quantities against the step's configured input-to-output ratio; when several parts are supplied, each must work out to the same output quantity or the move is rejected. The source batches are linked as inputs and closed, the step's material consumption runs asynchronously afterwards, and the new batch is closed immediately if the target step is the last one in the flow. Returns the newly created batch.
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
		ObjectType:          constants.ObjectTypeBatch,
		// A batch reports three measures; the units they are counted in are records of their own.
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeBatch,
			Fields:     []string{"quantity.unit", "seconds.unit", "waste.unit"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *MoveBatchesRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).MoveBatches
		},
	})
}
