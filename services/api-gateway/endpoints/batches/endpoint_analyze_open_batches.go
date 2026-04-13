package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to analyze open batches.
type AnalyzeOpenBatchesRequest struct {
	// Item IDs to filter by.
	ItemIDs []string `json:"item_ids"`
	// Product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids"`
}

var sampleAnalyzeOpenBatchesRequest = &AnalyzeOpenBatchesRequest{
	ItemIDs:        []string{apiresource.SampleItemID},
	ProductLineIDs: []string{apiresource.SampleProductLineID},
}

func (*AnalyzeOpenBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAnalyzeOpenBatchesRequest)
}

type AnalyzeOpenBatchesEndpoint struct{}

func (e *AnalyzeOpenBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.List[apiresource.OpenBatchSummary]] {
	return &apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.List[apiresource.OpenBatchSummary]]{
		Title:             "Analyze Open Batches",
		Description:       "Returns aggregated summaries of open batches, optionally filtered by item IDs or product line IDs.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/analytics/open-batches",
		Request:           &AnalyzeOpenBatchesRequest{},
		Response:          &apiresource.List[apiresource.OpenBatchSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.List[apiresource.OpenBatchSummary], *apierror.APIError) {
			return svc.(BatchSvc).AnalyzeOpenBatches
		},
	}
}
