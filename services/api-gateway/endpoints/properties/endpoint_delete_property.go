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
	"github.com/open-mrp/apikit/object"
)

// Request to delete a property.
type DeletePropertyRequest struct {
	// Property ID.
	PropertyID string `path:"id" validate:"required"`
}

// Deletes a property and its attributes.
//
// Only while no item category, item or price-list rule uses it or any of its attributes.
type DeletePropertyEndpoint struct{}

func (e *DeletePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePropertyRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeletePropertyRequest, *object.Deleted]{
		Title:             "Delete Property",
		Method:            http.MethodDelete,
		Route:             CatalogPropertyRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeProperty,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePropertyRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PropertySvc).DeleteProperty, constants.ObjectTypeProperty, "id")
		},
	})
}
