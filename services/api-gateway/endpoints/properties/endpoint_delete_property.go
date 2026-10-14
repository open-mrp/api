package propertyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
		Title:             "Delete Property",
		Method:            http.MethodDelete,
		Route:             CatalogPropertyRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PropertySvc).DeleteProperty
		},
	})
}
