package itemep

import (
	"context"
	"maps"
	"net/http"

	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

const (
	itemsRoute = "/v1/catalog/items"
	itemRoute  = "/v1/catalog/items/{id}"
)

// ItemIncludeKeys are the include keys every item endpoint offers.
var ItemIncludeKeys = []string{
	"category", "category.properties", "category.unit_group", "category.unit_group.base_unit",
	"category.unit_group.associated_units", "category.unit_group.associated_units.unit",
	"attributes", "attributes.property", "sales.product_line", "default_manufacturing_method",
}

func itemIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{ObjectType: constants.ObjectTypeItem, Fields: ItemIncludeKeys})
}

// Request to list items.
type ListItemsRequest struct {
	object.ListRequest
	apirequest.CreatedRange
	// Case-insensitive substring match on `sku` or `description`.
	//
	// Ranked: exact SKU first, then a SKU containing the term as a whole word, then a SKU starting with it, then all other matches. Newest first within each rank.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return items with exactly these SKUs (case-insensitive). At most 100.
	SKUs []string `query:"skus" validate:"max=100"`
	// Only return items of these types.
	Types []constants.ItemTypeCode `query:"types"`
	// Only return items with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
	// Only return items in one of these categories.
	CategoryIDs []string `query:"category_ids"`
	// Only return items carrying at least one of these attributes.
	AttributeIDs []string `query:"attribute_ids"`
	// Only return materials one of these suppliers supplies to you.
	SupplierIDs []string `query:"supplier_ids"`
	// Only return sold items whose product line is one of these.
	ProductLineIDs []string `query:"product_line_ids"`
	// Only return sellable items at least one of these customers may order, through access to the item's product line.
	CustomerIDs []string `query:"customer_ids"`
	// Only return sold items with one of these portal visibilities.
	PortalVisibilities []constants.CustomerPortalVisibility `query:"portal_visibilities"`
	// Restricts results by where the item is produced.
	//
	// `all`: no restriction. `initial_only`: only items produced by an operation nothing upstream feeds into.
	SubassemblyFilter *constants.SubassemblyFilter `query:"subassembly_filter" default:"all"`
}

var _ apiexample.Documented = (*ListItemsRequest)(nil)

// SchemaExample aligns list filters with SampleItem for OpenAPI documentation.
func (*ListItemsRequest) SchemaExample() any {
	base := apiresource.ListRequestSchemaExample()
	out := maps.Clone(base)
	out["types"] = []any{string(constants.ItemTypeCodeProduct)}
	out["skus"] = []any{apiresource.SampleItemSKU}
	return out
}

// Returns a list of items.
//
// Sorted by creation date, newest first, or by search rank when `q` is set.
type ListItemsEndpoint struct{}

func (e *ListItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListItemsRequest, *apiresource.List[apiresource.Item]] {
	return (&apiendpoint.APIEndpoint[*ListItemsRequest, *apiresource.List[apiresource.Item]]{
		Title:             "List Items",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             itemsRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
			return svc.(ItemSvc).ListItems
		},
		ObjectType:    constants.ObjectTypeItem,
		IncludeConfig: itemIncludes(),
	})
}
