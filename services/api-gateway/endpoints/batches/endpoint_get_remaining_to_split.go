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

// Request to get the remaining quantity available to split from batches.
type GetRemainingQuantityToSplitRequest struct {
	// Batch IDs to check remaining quantities for.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// The production step the split would be performed at; its configuration determines the expected output quantity and unit.
	ProductionStepID string `json:"production_step_id" validate:"required"`
}

var sampleGetRemainingQuantityToSplitRequest = &GetRemainingQuantityToSplitRequest{
	BatchIDs:         []string{apiresource.SampleBatchID},
	ProductionStepID: apiresource.SampleProductionStepID,
}

func (*GetRemainingQuantityToSplitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleGetRemainingQuantityToSplitRequest)
}

// Returns the remaining quantity available to split from the specified batches at a given production step.
//
// The remaining quantity is the step's expected output for the source batches minus the quantities already split off into output batches, expressed in the step's produced unit.
type GetRemainingQuantityToSplitEndpoint struct{}

func (e *GetRemainingQuantityToSplitEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetRemainingQuantityToSplitRequest, *apiresource.Quantity] {
	return (&apiendpoint.APIEndpoint[*GetRemainingQuantityToSplitRequest, *apiresource.Quantity]{
		Title:               "Get Remaining Quantity to Split",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/remaining-quantities",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetRemainingQuantityToSplitRequest) (*apiresource.Quantity, *apierror.APIError) {
			return svc.(BatchSvc).GetRemainingQuantityToSplit
		},
	})
}
