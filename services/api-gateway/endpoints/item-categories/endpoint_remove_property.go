package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove a property from an item category.
type RemoveItemCategoryPropertyRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// ID of the property to remove from the category.
	PropertyID string `path:"property_id" validate:"required"`
}

// Detaches a property from an item category.
//
// Only the link between the property and the category is removed; the property itself and its attributes are left intact and stay available to other categories. The property must belong to your account.
type RemoveItemCategoryPropertyEndpoint struct{}

func (e *RemoveItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveItemCategoryPropertyRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveItemCategoryPropertyRequest, *apiresource.EmptyResource]{
		Title:               "Remove Item Category Property",
		Method:              http.MethodDelete,
		Route:               "/v1/catalog/item-categories/{id}/properties/{property_id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).RemoveItemCategoryProperty
		},
	})
}
