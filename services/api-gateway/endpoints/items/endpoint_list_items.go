package itemep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListItemsRequest is the request to list items.
type ListItemsRequest struct {
	apiresource.PaginationRequest
	// Filter by item type codes.
	Types []string `query:"types"`
	// Filter by category IDs.
	CategoryIDs []string `query:"category_ids"`
	// Filter by attribute IDs.
	AttributeIDs []string `query:"attribute_ids"`
	// Filter by supplier ID.
	SupplierID *string `query:"supplier_id"`
	// Filter items created on or after this date.
	StartDate *time.Time `query:"start_date"`
	// Filter items created on or before this date.
	EndDate *time.Time `query:"end_date"`
	// How the search query is matched against items (default: partial).
	MatchMode *constants.ItemMatchMode `query:"match_mode" default:"partial"`
	// Which subassemblies to include when listing (default: all).
	SubassemblyFilter *constants.SubassemblyFilter `query:"subassembly_filter" default:"all"`
	// Filter by product line IDs (only items whose product belongs to one of these lines).
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by customer account IDs (only items whose product line is accessible to any of these customers).
	CustomerIDs []string `query:"customer_ids"`
}

// Returns a paginated list of items.
type ListItemsEndpoint struct{}

func (e *ListItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListItemsRequest, *apiresource.List[apiresource.Item]] {
	return (&apiendpoint.APIEndpoint[*ListItemsRequest, *apiresource.List[apiresource.Item]]{
		Title:             "List Items",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
			return svc.(ItemSvc).ListItems
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	})
}
