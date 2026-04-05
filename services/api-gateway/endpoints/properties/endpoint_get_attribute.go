package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetAttributeRequest is the request to retrieve a single attribute.
type GetAttributeRequest struct {
	// The ID of the property.
	PropertyID string `path:"property_id" validate:"required"`
	// The ID of the attribute to retrieve.
	AttributeID string `path:"id" validate:"required"`
}

type GetAttributeEndpoint struct{}

func (e *GetAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAttributeRequest, *apiresource.Attribute] {
	return &apiendpoint.APIEndpoint[*GetAttributeRequest, *apiresource.Attribute]{
		Title:             "Get Attribute",
		Description:       "Returns a single attribute by its ID within a property.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/properties/{property_id}/attributes/{id}",
		Request:           &GetAttributeRequest{},
		Response:          &apiresource.Attribute{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).GetAttribute
		},
	}
}
