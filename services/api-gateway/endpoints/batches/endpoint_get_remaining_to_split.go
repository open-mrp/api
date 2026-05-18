package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get the remaining quantity available to split from batches.
type GetRemainingQuantityToSplitRequest struct {
	// Batch IDs to check remaining quantities for.
	BatchIDs []string `json:"batch_ids" validate:"required"`
	// Production step ID to check against.
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
type GetRemainingQuantityToSplitEndpoint struct{}

func (e *GetRemainingQuantityToSplitEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetRemainingQuantityToSplitRequest, *apiresource.Quantity] {
	return (&apiendpoint.APIEndpoint[*GetRemainingQuantityToSplitRequest, *apiresource.Quantity]{
		Title:             "Get Remaining Quantity to Split",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/remaining-quantities",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetRemainingQuantityToSplitRequest) (*apiresource.Quantity, *apierror.APIError) {
			return svc.(BatchSvc).GetRemainingQuantityToSplit
		},
	})
}
