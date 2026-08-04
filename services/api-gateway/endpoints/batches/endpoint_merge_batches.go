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

// Request to merge multiple batches into one.
type MergeBatchesRequest struct {
	// Batch IDs to merge.
	//
	// Duplicates are rejected. For single-part production steps all batches must be of the same item; for multi-part steps supply at least one batch per part the step consumes. Each ID is resolved forward through its production flow to the batch that is actually available at the step, so an operator can scan an earlier batch in the chain.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Scanning station ID performing the merge.
	ScanningStationID string `json:"scanning_station_id" validate:"required"`
	// The production step the merged batch is created at.
	ProductionStepID string `json:"production_step_id" validate:"required"`
}

var sampleMergeBatchesRequest = &MergeBatchesRequest{
	BatchIDs:          []string{apiresource.SampleBatchID},
	ScanningStationID: apiresource.SampleScanningStationID,
	ProductionStepID:  apiresource.SampleProductionStepID,
}

func (*MergeBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMergeBatchesRequest)
}

// Merges multiple batches into a single new batch at a production step.
//
// The new batch is created at the target step and its quantity is scaled from how much input was supplied against the step's configured input-to-output ratio: for a single-part step the source quantities are summed, and for a multi-part step every part must work out to the same output quantity or the merge is rejected. The source batches are linked as inputs and closed, the step's material consumption runs asynchronously afterwards, and the new batch is closed immediately if the target step is the last one in the flow. Returns the newly created batch.
type MergeBatchesEndpoint struct{}

func (e *MergeBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*MergeBatchesRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*MergeBatchesRequest, *apiresource.Batch]{
		Title:               "Merge Batches",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/actions/merge",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MergeBatchesRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).MergeBatches
		},
	})
}
