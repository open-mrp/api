package analyticsep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// AnalyzeOpenBatchesRequest is the request to analyze open batches.
type AnalyzeOpenBatchesRequest struct {
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
}

// Returns open batch summaries grouped by scanning station and item.
//
// Each open, scanned batch counts for its quantity less what has already gone downstream into output batches. Selecting items or product lines selects the parts their production consumes, recursively, together with the selected items themselves; a selection that leads to no part does not filter.
type AnalyzeOpenBatchesEndpoint struct{}

func (e *AnalyzeOpenBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.AnalyzeOpenBatchesResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.AnalyzeOpenBatchesResponse]{
		Title:             "Analyze Open Batches",
		Method:            http.MethodPost,
		Route:             "/v1/core/analytics/open-batches/actions/query",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ReadOnly:          true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.AnalyzeOpenBatchesResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOpenBatches
		},
	})
}
