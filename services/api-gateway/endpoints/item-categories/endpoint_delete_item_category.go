package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete an item category.
type DeleteItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
}

// Deletes an item category owned by your account.
//
// System-owned categories cannot be deleted. Deleting a category that was already deleted returns an already-deleted error rather than a not-found error.
type DeleteItemCategoryEndpoint struct{}

func (e *DeleteItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteItemCategoryRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteItemCategoryRequest, *apiresource.EmptyResource]{
		Title:               "Delete Item Category",
		Method:              http.MethodDelete,
		Route:               "/v1/catalog/item-categories/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteItemCategoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).DeleteItemCategory
		},
	})
}
