package propertyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete an attribute.
type DeleteAttributeRequest struct {
	// The property the attribute belongs to.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
}

// Deletes an attribute.
//
// Only while no item or price-list rule uses it. Archive it instead by setting `status` to `archived`. Later attributes shift up to keep `sort_order` contiguous.
type DeleteAttributeEndpoint struct{}

func (e *DeleteAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAttributeRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteAttributeRequest, *object.Deleted]{
		Title:             "Delete Attribute",
		Method:            http.MethodDelete,
		Route:             CatalogPropertyAttributeRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeAttribute,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAttributeRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PropertySvc).DeleteAttribute, constants.ObjectTypeAttribute, "id")
		},
	})
}
