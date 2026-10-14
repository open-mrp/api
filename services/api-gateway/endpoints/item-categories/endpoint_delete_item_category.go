package itemcategoryep

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

// Request to delete an item category.
type DeleteItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
}

// Deletes an item category.
//
// Only categories your account created can be deleted, and only while no item or price-list rule uses them. Move a category's items to another category before deleting it.
type DeleteItemCategoryEndpoint struct{}

func (e *DeleteItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteItemCategoryRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteItemCategoryRequest, *object.Deleted]{
		Title:             "Delete Item Category",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/item-categories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeItemCategory,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteItemCategoryRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ItemCategorySvc).DeleteItemCategory, constants.ObjectTypeItemCategory, "id")
		},
	})
}
