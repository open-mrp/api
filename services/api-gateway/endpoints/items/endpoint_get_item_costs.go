package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetItemCostsRequest is the request to retrieve costs for an item.
type GetItemCostsRequest struct {
	// The ID of the item to retrieve costs for.
	ItemID string `path:"id" validate:"required"`
}

type GetItemCostsEndpoint struct{}

func (e *GetItemCostsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemCostsRequest, *apiresource.ItemCosts] {
	return &apiendpoint.APIEndpoint[*GetItemCostsRequest, *apiresource.ItemCosts]{
		Title:             "Get Item Costs",
		Description:       "Returns the production cost breakdown for an item, including direct material, direct labor, overhead, and total costs.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/items/{id}/costs",
		Request:           &GetItemCostsRequest{},
		Response:          &apiresource.ItemCosts{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemCostsRequest) (*apiresource.ItemCosts, *apierror.APIError) {
			return svc.(ItemSvc).GetItemCosts
		},
	}
}
