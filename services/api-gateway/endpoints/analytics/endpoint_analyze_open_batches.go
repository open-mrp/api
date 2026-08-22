package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// AnalyzeOpenBatchesRequest is the request to analyze open batches.
type AnalyzeOpenBatchesRequest struct {
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
}

// Returns open batch summaries grouped by scanning station.
type AnalyzeOpenBatchesEndpoint struct{}

func (e *AnalyzeOpenBatchesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.AnalyzeOpenBatchesResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeOpenBatchesRequest, *apiresource.AnalyzeOpenBatchesResponse]{
		Title:               "Analyze Open Batches",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/open-batches",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.AnalyzeOpenBatchesResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOpenBatches
		},
	})
}
