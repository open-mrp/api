package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
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

// Returns the account's available inventory receipts grouped by item, location, lot, owner and holder, oldest receipt first.
//
// Each receipt counts for what is left of it after its allocations, in the item's base unit and never below zero; a receipt without a unit cost is left out. The weighted average unit cost is per base unit, and the inventory value is the remaining quantity at that cost, in the oldest receipt's currency. A group with nothing left has no inventory value and a weighted average cost of zero.
type AnalyzeInventoryReceiptsEndpoint struct{}

func (e *AnalyzeInventoryReceiptsEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeInventoryReceiptsRequest, *apiresource.AnalyzeInventoryReceiptsResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeInventoryReceiptsRequest, *apiresource.AnalyzeInventoryReceiptsResponse]{
		Title:                   "Analyze Inventory Receipts",
		Method:                  http.MethodPut,
		Route:                   "/v1/core/analytics/inventory-receipts",
		ContentType:             "application/json",
		SuccessStatusCode:       http.StatusOK,
		Public:                  false,
		AgentTool:               true,
		CounterpartyPermissions: apiendpoint.Counterparties(types.ActionRead),
		Preview:                 true,
		RequiredPermissions:     []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeInventoryReceiptsRequest) (*apiresource.AnalyzeInventoryReceiptsResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeInventoryReceipts
		},
	})
}
