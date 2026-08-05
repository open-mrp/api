package itemep

import (
	"context"
	"maps"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list items.
type ListItemsRequest struct {
	apiresource.PaginationRequest
	// Filter to items of these types (`product`, `material`, `part`).
	Types []string `query:"types"`
	// Filter to items in any of these categories.
	CategoryIDs []string `query:"category_ids"`
	// Filter to items carrying any of these attributes.
	AttributeIDs []string `query:"attribute_ids"`
	// Filter to materials this supplier account supplies to you.
	//
	// Only materials can have suppliers, so combining this with a `types` filter that excludes `material` returns nothing.
	SupplierID *string `query:"supplier_id"`
	// Filter to items created on or after this date.
	StartDate *time.Time `query:"starts_at"`
	// Filter to items created on or before this date.
	EndDate *time.Time `query:"ends_at"`
	// Restricts results based on where the item is produced in its production flow.
	//
	// - `all`: no restriction.
	// - `initial_only`: only items produced by an initial production step, i.e. a step with no upstream steps feeding into it.
	SubassemblyFilter *constants.SubassemblyFilter `query:"subassembly_filter" default:"all"`
	// Filter to items whose product belongs to any of these product lines.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter to items any of these customers are allowed to order.
	//
	// A customer qualifies when its relationship, its account group, or its price group grants access to the product line the item's product sits in. Items with no product line, including materials and parts, never match.
	CustomerIDs []string `query:"customer_ids"`
}

var _ contracts.DocumentedType = (*ListItemsRequest)(nil)

// SchemaExample aligns list filters with SampleItem for OpenAPI documentation.
func (*ListItemsRequest) SchemaExample() any {
	base := (&apiresource.PaginationRequest{}).SchemaExample().(map[string]any)
	out := maps.Clone(base)
	out["types"] = []any{string(constants.ItemTypeCodeProduct)}
	return out
}

// Returns a paginated list of items, newest first.
//
// Items backed by a non-sale product — the service, shipping, tax, credit, and return products that carry charges on orders — are left out, so this reflects the catalog you sell and stock rather than every item row. `q` matches against SKU and description, with closer SKU matches ranked first.
type ListItemsEndpoint struct{}

func (e *ListItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListItemsRequest, *apiresource.List[apiresource.Item]] {
	return (&apiendpoint.APIEndpoint[*ListItemsRequest, *apiresource.List[apiresource.Item]]{
		Title:               "List Items",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
			return svc.(ItemSvc).ListItems
		},
		ObjectType: constants.ObjectTypeItem,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	})
}
