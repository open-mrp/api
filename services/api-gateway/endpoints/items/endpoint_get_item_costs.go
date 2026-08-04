package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an item's cost breakdown.
type GetItemCostsRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Returns what it costs to make one unit of an item, split into direct material, direct labor, and overhead.
//
// The figures are recomputed on each call by walking back through every production step that feeds the step producing this item, so the answer reflects the current recipe and the current cost of everything consumed along the way. Items that no production flow produces — purchased materials, for instance — return a not-found error rather than a zero breakdown.
//
// Calling this also writes the computed total back to the item's `unit_cost`, so it is how a stale unit cost gets refreshed.
type GetItemCostsEndpoint struct{}

func (e *GetItemCostsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemCostsRequest, *apiresource.ItemCosts] {
	return (&apiendpoint.APIEndpoint[*GetItemCostsRequest, *apiresource.ItemCosts]{
		Title:               "Get Item Costs",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/costs",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemCostsRequest) (*apiresource.ItemCosts, *apierror.APIError) {
			return svc.(ItemSvc).GetItemCosts
		},
	})
}
