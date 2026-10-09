package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an item's cost breakdown.
type GetItemCostsRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Returns what it costs to make one unit of an item, split into direct material, direct labor, and overhead.
//
// The figures are recomputed on each call by walking back through every production step that feeds the step producing this item, so the answer reflects the current recipe and the current cost of everything consumed along the way. Each upstream step is scaled by how much of its output the step after it consumes, both sides converted through their units, so a step drawing 2 dozen from one that produces 12 eaches is costed as two of its runs. Items that no production flow produces — purchased materials, for instance — return a not-found error rather than a zero breakdown, as do items whose producing step has a zero production quantity.
//
// Reading costs stores nothing. The item's `unit_cost` is restated separately, shortly after anything it is built from changes — a material's cost, a production step, a consumption.
type GetItemCostsEndpoint struct{}

func (e *GetItemCostsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemCostsRequest, *apiresource.ItemCosts] {
	return (&apiendpoint.APIEndpoint[*GetItemCostsRequest, *apiresource.ItemCosts]{
		Title:               "Get Item Costs",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/costs",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCosts, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemCostsRequest) (*apiresource.ItemCosts, *apierror.APIError) {
			return svc.(ItemSvc).GetItemCosts
		},
	})
}
