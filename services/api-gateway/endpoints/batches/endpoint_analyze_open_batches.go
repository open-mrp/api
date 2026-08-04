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

// Request to analyze open batches.
type AnalyzeOpenBatchesRequest struct {
	// Restrict the summaries to batches of these items; omit to include all items.
	ItemIDs []string `json:"item_ids"`
	// Restrict the summaries to batches whose item belongs to these product lines; omit to include all product lines.
	ProductLineIDs []string `json:"product_line_ids"`
}

var sampleAnalyzeOpenBatchesRequest = &AnalyzeOpenBatchesRequest{
	ItemIDs:        []string{apiresource.SampleItemID},
	ProductLineIDs: []string{apiresource.SampleProductLineID},
}

func (*AnalyzeOpenBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAnalyzeOpenBatchesRequest)
}

// Returns the work in progress currently sitting on the production floor, grouped by department, item, and scanning station.
//
// Only batches that have been scanned at a scanning station and are not yet closed are counted, and each batch contributes its quantity less whatever has already moved downstream, so the totals show what is still left to work on. Each result covers one item at one scanning station.
type AnalyzeOpenBatchesEndpoint struct{}

func (e *AnalyzeOpenBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.List[apiresource.OpenBatchSummary]] {
	return (&apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.List[apiresource.OpenBatchSummary]]{
		Title:               "Get Open Batch Summaries",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/operations/analytics/open-batches",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.List[apiresource.OpenBatchSummary], *apierror.APIError) {
			return svc.(BatchSvc).AnalyzeOpenBatches
		},
	})
}
