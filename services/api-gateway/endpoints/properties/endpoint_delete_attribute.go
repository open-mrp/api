package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an attribute.
type DeleteAttributeRequest struct {
	// Property ID.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
}

// Deletes an attribute from a property.
type DeleteAttributeEndpoint struct{}

func (e *DeleteAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAttributeRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAttributeRequest, *apiresource.EmptyResource]{
		Title:             "Delete Attribute",
		Method:            http.MethodDelete,
		Route:             CatalogPropertyAttributeRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAttributeRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PropertySvc).DeleteAttribute
		},
	})
}
