package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list attributes for a property.
type ListAttributesRequest struct {
	apiresource.PaginationRequest
	// Property ID.
	PropertyID string `path:"property_id" validate:"required"`
}

// Returns a paginated list of attributes for a property.
type ListAttributesEndpoint struct{}

func (e *ListAttributesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAttributesRequest, *apiresource.List[apiresource.Attribute]] {
	return (&apiendpoint.APIEndpoint[*ListAttributesRequest, *apiresource.List[apiresource.Attribute]]{
		Title:               "List Attributes",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               CatalogPropertyAttributesRoute,
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAttribute,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAttributesRequest) (*apiresource.List[apiresource.Attribute], *apierror.APIError) {
			return svc.(PropertySvc).ListAttributes
		},
	})
}
