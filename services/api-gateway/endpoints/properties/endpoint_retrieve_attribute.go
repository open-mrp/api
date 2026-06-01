package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an attribute.
type RetrieveAttributeRequest struct {
	// Property ID.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
}

// Returns an attribute by ID within a property.
type RetrieveAttributeEndpoint struct{}

func (e *RetrieveAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAttributeRequest, *apiresource.Attribute] {
	return (&apiendpoint.APIEndpoint[*RetrieveAttributeRequest, *apiresource.Attribute]{
		Title:             "Retrieve Attribute",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             CatalogPropertyAttributeRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAttribute,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).GetAttribute
		},
	})
}
