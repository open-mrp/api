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
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// The property whose attributes are listed.
	PropertyID string `path:"property_id" validate:"required"`
}

// Returns a paginated list of attributes for a property.
//
// Attributes come back in the order they are arranged within the property, first to last. The `q` search term is matched against the attribute value.
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
		Preview:           true,
		ObjectType:        constants.ObjectTypeAttribute,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAttributesRequest) (*apiresource.List[apiresource.Attribute], *apierror.APIError) {
			return svc.(PropertySvc).ListAttributes
		},
	})
}
