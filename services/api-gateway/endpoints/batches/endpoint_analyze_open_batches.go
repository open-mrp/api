package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeOpenBatchesRequest is the request to analyze open batches filtered by items or product lines.
type AnalyzeOpenBatchesRequest struct {
	// Optional list of item IDs to filter by.
	ItemIDs []string `json:"item_ids"`
	// Optional list of product line IDs to filter by.
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
		Title:             "Analyze Open Batch Summaries",
		Description:       "Returns aggregated summaries of open batches, optionally filtered by item IDs or product line IDs.",
		Method:            http.MethodPut,
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
