package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an attribute.
type RetrieveAttributeRequest struct {
	// The property the attribute belongs to.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
}

// Returns an attribute by ID within a property.
type RetrieveAttributeEndpoint struct{}

func (e *RetrieveAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAttributeRequest, *apiresource.Attribute] {
	return (&apiendpoint.APIEndpoint[*RetrieveAttributeRequest, *apiresource.Attribute]{
		Title:               "Retrieve Attribute",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               CatalogPropertyAttributeRoute,
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAttribute,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).GetAttribute
		},
	})
}
