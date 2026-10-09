package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve trend data for an item.
type GetItemTrendsRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// The trend metric to fetch.
	//
	// `inventory` returns the item's logged inventory level at the close of each of the last 30 days.
	TrendType constants.ItemTrendType `query:"trend_type" validate:"required"`
}

// Returns how an item's stock level has moved over the last 30 days: exactly 30 points, one per UTC calendar day ending today, oldest first.
//
// Each point is the last inventory level logged on or before the end of its day, so a day with several movements reports where it closed. A day with nothing logged carries the previous day's level forward, and the series opens at the last level logged before the window rather than at zero, so an item that has not moved for a month shows its stock rather than a flat line at zero. An item never logged reads zero throughout.
//
// A logged level is the item's physical stock — on hand less what is short against open demand — at the moment of the movement that wrote it. Every value is converted into the base unit of the item's category, returned as `unit`, whatever unit the movement was recorded in.
type GetItemTrendsEndpoint struct{}

func (e *GetItemTrendsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemTrendsRequest, *apiresource.ItemTrends] {
	return (&apiendpoint.APIEndpoint[*GetItemTrendsRequest, *apiresource.ItemTrends]{
		Title:               "Get Item Trends",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/trends",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemTrendsRequest) (*apiresource.ItemTrends, *apierror.APIError) {
			return svc.(ItemSvc).GetItemTrends
		},
	})
}
