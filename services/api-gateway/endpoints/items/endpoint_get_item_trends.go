package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve trend data for an item.
type GetItemTrendsRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// The trend metric to fetch.
	//
	// Currently the only supported value is `inventory`, which returns the item's inventory-level measurements from the last 30 days. Unsupported values are rejected with a validation error.
	TrendType string `query:"trend_type" validate:"required"`
}

// Returns how an item's stock level has moved over the last 30 days, as a series of point-in-time measurements.
//
// Days on which nothing was logged produce no point, and days with several entries contribute only the first, so the series is sparse rather than one point per calendar day.
type GetItemTrendsEndpoint struct{}

func (e *GetItemTrendsEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemTrendsRequest, *apiresource.ItemTrends] {
	return (&apiendpoint.APIEndpoint[*GetItemTrendsRequest, *apiresource.ItemTrends]{
		Title:               "Get Item Trends",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/trends",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemTrendsRequest) (*apiresource.ItemTrends, *apierror.APIError) {
			return svc.(ItemSvc).GetItemTrends
		},
	})
}
