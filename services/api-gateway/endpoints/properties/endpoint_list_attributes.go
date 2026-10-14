package propertyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list attributes for a property.
type ListAttributesRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `value`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return attributes with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
	// The property whose attributes are listed.
	PropertyID string `path:"property_id" validate:"required"`
}

// Returns a list of a property's attributes.
//
// Sorted by `sort_order`, first to last.
type ListAttributesEndpoint struct{}

func (e *ListAttributesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAttributesRequest, *apiresource.List[apiresource.Attribute]] {
	return (&apiendpoint.APIEndpoint[*ListAttributesRequest, *apiresource.List[apiresource.Attribute]]{
		Title:             "List Attributes",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             CatalogPropertyAttributesRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeAttribute,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAttribute,
			Fields:     []string{"property"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAttributesRequest) (*apiresource.List[apiresource.Attribute], *apierror.APIError) {
			return svc.(PropertySvc).ListAttributes
		},
	})
}
