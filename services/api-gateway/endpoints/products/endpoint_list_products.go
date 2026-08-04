package productep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list products.
type ListProductsRequest struct {
	apiresource.PaginationRequest
	// Restrict results to products these customer accounts are entitled to buy.
	//
	// A product matches when its product line has been granted to the customer directly, through the customer's account group, or through the account group used for the customer's pricing. Combined with `product_line_ids` this widens the results rather than narrowing them: products matching either filter are returned.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by product line IDs.
	//
	// Combined with `customer_ids`, products matching either filter are returned.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by the item category the product's item belongs to.
	CategoryIDs []string `query:"category_ids"`
	// Filter to products whose item carries at least one of these attributes.
	AttributeIDs []string `query:"attribute_ids"`
	// Start of creation date range.
	StartDate *time.Time `query:"start_date"`
	// End of creation date range.
	EndDate *time.Time `query:"end_date"`
	// Filter by customer portal visibility.
	PortalVisibility *constants.CustomerPortalVisibility `query:"portal_visibility"`
}

// Returns a paginated list of products for the target account, newest first.
//
// Only products of type `sale` are listed — service, shipping, credit, return, and tax products are excluded and must be retrieved by ID. A request made by a customer-portal buyer always returns portal-visible products only, and its `customer_ids` filter is replaced with the buyer's own account, so the results reflect what that account is entitled to buy.
//
// The `q` search term is matched against the SKU and description of each product's item; when it is supplied, products whose SKU matches are returned ahead of the rest.
type ListProductsEndpoint struct{}

func (e *ListProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductsRequest, *apiresource.List[apiresource.Product]] {
	return (&apiendpoint.APIEndpoint[*ListProductsRequest, *apiresource.List[apiresource.Product]]{
		Title:               "List Products",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/products",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductsRequest) (*apiresource.List[apiresource.Product], *apierror.APIError) {
			return svc.(ProductSvc).ListProducts
		},
		ObjectType: constants.ObjectTypeProduct,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
