package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteAttributeRequest is the request to delete an attribute.
type DeleteAttributeRequest struct {
	// The ID of the property.
	PropertyID string `path:"property_id" validate:"required"`
	// The ID of the attribute to delete.
	AttributeID string `path:"id" validate:"required"`
}

type DeleteAttributeEndpoint struct{}

func (e *DeleteAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAttributeRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteAttributeRequest, *apiresource.EmptyResource]{
		Title:             "Delete Attribute",
		Description:       "Deletes an attribute from a property.",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/properties/{property_id}/attributes/{id}",
		ContentType:       "application/json",
		Request:           &DeleteAttributeRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAttributeRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PropertySvc).DeleteAttribute
		},
	}
}
