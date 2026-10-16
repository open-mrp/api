package searchep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to search across resource types for a free-text term.
type SearchRequest struct {
	object.ListRequest
	// The search term. Each type matches its own columns:
	//
	// - `sales_order`: exactly the order `number` or the customer's PO number.
	// - `purchase_order`: substring of the order `number`, the supplier's name, the PO number, or the supplier's external number, alias or notes.
	// - `invoice`: substring of the invoice `number` or note, the customer's name, the order number or PO number, or the customer's external number, alias or notes.
	// - `customer`: substring of `name` or `number`.
	// - `item` and `product`: substring of the item's `sku` or `description`, exact SKU matches first.
	// - `shipment`: substring of the shipment `number`, note, bill of lading or master tracking number, the order number or PO number, or the customer's name.
	// - `account_user`: substring of the person's name.
	// - `agent_definition`: substring of the agent's ID, `name`, `description` or `slug`.
	//
	// Matching is case-insensitive. Required unless the search is narrowed with `types`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only search these resource types: `sales_order`, `purchase_order`, `invoice`, `customer`, `item`, `product`, `shipment`, `account_user` or `agent_definition`.
	//
	// Any other value is rejected. Types you lack read permission for are dropped rather than rejected, so narrowing to a type you can't read returns no results. Omit to search every searchable type you can read.
	Types []constants.ObjectType `query:"types"`
	// Only return records belonging to any of these customers.
	//
	// When set, only sales orders, invoices and shipments are searched, so a result can never point at a record those customers may not see.
	CustomerIDs []string `query:"customer_ids"`
}

// Searches several resource types at once.
//
// Each result names the matched record's type and ID; include `resource` to get the record itself.
//
// `q` is required unless the search is narrowed with `types`, which lets you omit `q` to browse that type's most recent records. Matches are drawn from every searchable type you can read, then interleaved so no single type crowds out the others, and the combined result set is capped at `limit`. Results are not paginated: `limit` is the total you get. If one resource type fails to respond, it contributes no results instead of failing the whole search.
type SearchEndpoint struct{}

func (e *SearchEndpoint) Materialize() *apiendpoint.APIEndpoint[*SearchRequest, *apiresource.List[apiresource.SearchResult]] {
	return (&apiendpoint.APIEndpoint[*SearchRequest, *apiresource.List[apiresource.SearchResult]]{
		Title:             "Search",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/search",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: searchReadPermissions},
		ObjectType:        constants.ObjectTypeSearchResult,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSearchResult,
			Fields:     []string{"resource", "entity"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *SearchRequest) (*apiresource.List[apiresource.SearchResult], *apierror.APIError) {
			return svc.(SearchSvc).Search
		},
	})
}
