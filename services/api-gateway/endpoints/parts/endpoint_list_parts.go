package partep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list parts.
type ListPartsRequest struct {
	apiresource.PaginationRequest
	// Only return parts belonging to any of these item categories.
	CategoryIDs []string `query:"category_ids"`
	// Only return parts carrying at least one of these attributes.
	AttributeIDs []string `query:"attribute_ids"`
	// Only return parts created at or after this time.
	StartDate *time.Time `query:"starts_at"`
	// Only return parts created at or before this time.
	EndDate *time.Time `query:"ends_at"`
}

// Returns a paginated list of parts for the current account, most recently created first.
//
// The `q` search term matches the part's SKU or description. When it is supplied, the parts whose SKU matches it most closely are returned first, ordered by creation time within each level of match.
type ListPartsEndpoint struct{}

func (e *ListPartsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPartsRequest, *apiresource.List[apiresource.Part]] {
	return (&apiendpoint.APIEndpoint[*ListPartsRequest, *apiresource.List[apiresource.Part]]{
		Title:               "List Parts",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/parts",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainParts, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPartsRequest) (*apiresource.List[apiresource.Part], *apierror.APIError) {
			return svc.(PartSvc).ListParts
		},
		ObjectType: constants.ObjectTypePart,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
