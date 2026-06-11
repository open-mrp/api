package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetItemCostsRequest is the request to get an item's cost breakdown.
type GetItemCostsRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Returns the per-unit production cost breakdown for an item, including direct material, direct labor, overhead, and total costs.
//
// Costs are computed from the production flow that produces the item; items not produced by any production flow return a not-found error. As a side effect, the item's `unit_cost` rate is refreshed to the computed total.
type GetItemCostsEndpoint struct{}

func (e *GetItemCostsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemCostsRequest, *apiresource.ItemCosts] {
	return (&apiendpoint.APIEndpoint[*GetItemCostsRequest, *apiresource.ItemCosts]{
		Title:             "Get Item Costs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/costs",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemCostsRequest) (*apiresource.ItemCosts, *apierror.APIError) {
			return svc.(ItemSvc).GetItemCosts
		},
	})
}
