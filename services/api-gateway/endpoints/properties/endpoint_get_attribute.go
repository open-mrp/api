package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an attribute.
type GetAttributeRequest struct {
	// Property ID.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
}

type GetAttributeEndpoint struct{}

func (e *GetAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAttributeRequest, *apiresource.Attribute] {
	return &apiendpoint.APIEndpoint[*GetAttributeRequest, *apiresource.Attribute]{
		Title:             "Get Attribute",
		Description:       "Returns an attribute by ID within a property.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
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
