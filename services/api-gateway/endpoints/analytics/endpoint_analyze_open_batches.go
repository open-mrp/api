package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeOpenBatchesRequest is the request to analyze open batches.
type AnalyzeOpenBatchesRequest struct {
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
}

type AnalyzeOpenBatchesEndpoint struct{}

func (e *AnalyzeOpenBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.AnalyzeOpenBatchesResponse] {
	return &apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.AnalyzeOpenBatchesResponse]{
		Title:             "Analyze Open Batches",
		Description:       "Returns open batch summaries grouped by scanning station.",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/open-batches",
		ContentType:       "application/json",
		Request:           &AnalyzeOpenBatchesRequest{},
		Response:          &apiresource.AnalyzeOpenBatchesResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.AnalyzeOpenBatchesResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOpenBatches
		},
	}
}
