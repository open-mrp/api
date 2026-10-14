package propertyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to delete an attribute.
type DeleteAttributeRequest struct {
	// The property the attribute belongs to.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
}

// Deletes an attribute from a property.
//
// Remaining attributes in the property are shifted so their sort orders stay contiguous.
type DeleteAttributeEndpoint struct{}

func (e *DeleteAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAttributeRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAttributeRequest, *apiresource.EmptyResource]{
		Title:             "Delete Attribute",
		Method:            http.MethodDelete,
		Route:             CatalogPropertyAttributeRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAttributeRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PropertySvc).DeleteAttribute
		},
	})
}
