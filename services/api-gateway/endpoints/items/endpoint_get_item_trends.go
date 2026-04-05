package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetItemTrendsRequest is the request to retrieve trend data for an item.
type GetItemTrendsRequest struct {
	// The ID of the item to retrieve trends for.
	ItemID string `path:"id" validate:"required"`
	// The type of trend to retrieve (e.g. "on_hand", "cost").
	TrendType string `query:"trend_type" validate:"required"`
}

type GetItemTrendsEndpoint struct{}

func (e *GetItemTrendsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemTrendsRequest, *apiresource.ItemTrends] {
	return &apiendpoint.APIEndpoint[*GetItemTrendsRequest, *apiresource.ItemTrends]{
		Title:             "Get Item Trends",
		Description:       "Returns historical trend data for an item for the specified metric.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/items/{id}/trends",
		Request:           &GetItemTrendsRequest{},
		Response:          &apiresource.ItemTrends{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemTrendsRequest) (*apiresource.ItemTrends, *apierror.APIError) {
			return svc.(ItemSvc).GetItemTrends
		},
	}
}
