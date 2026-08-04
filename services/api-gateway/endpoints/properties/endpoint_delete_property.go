package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a property.
type DeletePropertyRequest struct {
	// Property ID.
	PropertyID string `path:"id" validate:"required"`
}

// Deletes a property and every attribute defined under it.
//
// Items previously classified by those attributes lose that classification.
type DeletePropertyEndpoint struct{}

func (e *DeletePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePropertyRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeletePropertyRequest, *apiresource.EmptyResource]{
		Title:               "Delete Property",
		Method:              http.MethodDelete,
		Route:               CatalogPropertyRoute,
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PropertySvc).DeleteProperty
		},
	})
}
