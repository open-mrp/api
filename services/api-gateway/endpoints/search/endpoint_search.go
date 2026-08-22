package searchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to search across resource types for a free-text term.
type SearchRequest struct {
	apiresource.PaginationRequest
	// Filter the search to specific resource types.
	//
	// Only a subset of resource types is searchable: `sales_order`, `purchase_order`, `invoice`, `customer`, `item`, `product`, `shipment`, `messaging_contact`, and `agent_definition`. Any other value is rejected. Types you lack read permission for are silently dropped rather than rejected, so narrowing to a type you cannot read simply returns no results. Omit to search every searchable type you can read.
	Types []constants.ObjectType `query:"types"`
	// Restrict the search to a single customer by their account ID.
	//
	// When set, only resource types that are safe to expose to a customer are searched (their sales orders, invoices, and shipments), and results are limited to records belonging to that customer. This is intended for composing customer-facing replies, so a reference can never point at a record the customer is not entitled to see.
	Customer *string `query:"customer"`
}

// Searches across multiple resource types at once and returns lightweight `entity` references to the matches.
//
// Each result carries the matched record's ID, its resource type, and a display name and secondary handle, so it can be shown in a picker or turned into a link; fetch the record itself through its own endpoint for full detail.
//
// `q` is required unless the search is narrowed with `types`; scoping to one or more types lets you omit `q` to browse that type's most recent records. Matches are drawn from every searchable type you can read, then interleaved so no single type crowds out the others, and the combined result set is capped at `limit`. Results are not paginated — `limit` is the total you get. If one resource type fails to respond, it contributes no results instead of failing the whole search.
type SearchEndpoint struct{}

func (e *SearchEndpoint) Materialize() *apiendpoint.APIEndpoint[*SearchRequest, *apiresource.List[apiresource.Entity]] {
	return (&apiendpoint.APIEndpoint[*SearchRequest, *apiresource.List[apiresource.Entity]]{
		Title:               "Search",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/core/search",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		AgentTool:           true,
		RequiredPermissions: searchReadPermissions,
		ObjectType:          constants.ObjectTypeEntity,
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SearchRequest) (*apiresource.List[apiresource.Entity], *apierror.APIError) {
			return svc.(SearchSvc).Search
		},
	})
}
