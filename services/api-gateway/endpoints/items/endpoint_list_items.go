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

// ListItemsRequest is the request to list items with optional filters.
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
	// When true, search query must match exactly rather than partial match.
	IsExactMatch bool `query:"is_exact_match"`
	// When true, only return items that are initial subassemblies.
	OnlyInitialSubassemblies bool `query:"only_initial_subassemblies"`
}

type ListItemsEndpoint struct{}

func (e *ListItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListItemsRequest, *apiresource.List[apiresource.Item]] {
	return &apiendpoint.APIEndpoint[*ListItemsRequest, *apiresource.List[apiresource.Item]]{
		Title:             "List Items",
		Description:       "Returns a paginated list of items for the target account, with filtering by type, category, attributes, supplier, and date range.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items",
		Request:           &ListItemsRequest{},
		Response:          &apiresource.List[apiresource.Item]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
			return svc.(ItemSvc).ListItems
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate"},
		}),
	}
}
