package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetItemTrendsRequest is the request to get trend data for an item.
type GetItemTrendsRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// The trend metric to fetch.
	//
	// Currently the only supported value is `inventory`, which returns the item's inventory-level measurements from the last 30 days. Unsupported values are rejected with a validation error.
	TrendType string `query:"trend_type" validate:"required"`
}

// Returns historical trend data for an item as a time-ordered series of data points.
type GetItemTrendsEndpoint struct{}

func (e *GetItemTrendsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemTrendsRequest, *apiresource.ItemTrends] {
	return (&apiendpoint.APIEndpoint[*GetItemTrendsRequest, *apiresource.ItemTrends]{
		Title:             "Get Item Trends",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/trends",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemTrendsRequest) (*apiresource.ItemTrends, *apierror.APIError) {
			return svc.(ItemSvc).GetItemTrends
		},
	})
}
