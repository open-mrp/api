package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to add a property to an item category.
type AddItemCategoryPropertyRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// ID of the property to add to the category.
	PropertyID string `path:"property_id" validate:"required"`
}

// Adds a property to an item category, making the property available to items in that category.
//
// Each property name can appear only once per category; adding a property whose name duplicates one already in the category returns a conflict error. Default system categories cannot be modified.
type AddItemCategoryPropertyEndpoint struct{}

func (e *AddItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddItemCategoryPropertyRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*AddItemCategoryPropertyRequest, *apiresource.EmptyResource]{
		Title:               "Add Item Category Property",
		Method:              http.MethodPut,
		Route:               "/v1/catalog/item-categories/{id}/properties/{property_id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).AddItemCategoryProperty
		},
	})
}
