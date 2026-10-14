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
)

// Request to retrieve an attribute.
type RetrieveAttributeRequest struct {
	// The property the attribute belongs to.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
}

// Retrieves an attribute.
type RetrieveAttributeEndpoint struct{}

func (e *RetrieveAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAttributeRequest, *apiresource.Attribute] {
	return (&apiendpoint.APIEndpoint[*RetrieveAttributeRequest, *apiresource.Attribute]{
		Title:             "Retrieve Attribute",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             CatalogPropertyAttributeRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeAttribute,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAttribute,
			Fields:     []string{"property"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).GetAttribute
		},
	})
}
