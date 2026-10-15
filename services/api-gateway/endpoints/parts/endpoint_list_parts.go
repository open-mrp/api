package partep

import (
	"context"
	"net/http"
	"time"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list parts.
type ListPartsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
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
		Title:             "List Parts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/parts",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainParts, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPartsRequest) (*apiresource.List[apiresource.Part], *apierror.APIError) {
			return svc.(PartSvc).ListParts
		},
		ObjectType: constants.ObjectTypePart,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.attributes"},
		}),
	})
}
