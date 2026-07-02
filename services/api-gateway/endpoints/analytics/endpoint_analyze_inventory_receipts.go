package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeInventoryReceiptsRequest is the request to analyze inventory receipts.
type AnalyzeInventoryReceiptsRequest struct {
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
	// Optional location IDs to filter by.
	LocationIDs []string `json:"location_ids,omitempty"`
	// Optional lot IDs to filter by.
	LotIDs []string `json:"lot_ids,omitempty"`
}

// Returns inventory receipt summaries including remaining quantities, costs, and values.
type AnalyzeInventoryReceiptsEndpoint struct{}

func (e *AnalyzeInventoryReceiptsEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeInventoryReceiptsRequest, *apiresource.AnalyzeInventoryReceiptsResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeInventoryReceiptsRequest, *apiresource.AnalyzeInventoryReceiptsResponse]{
		Title:               "Analyze Inventory Receipts",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/inventory-receipts",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeInventoryReceiptsRequest) (*apiresource.AnalyzeInventoryReceiptsResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeInventoryReceipts
		},
	})
}
