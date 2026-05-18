package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an item category.
type DeleteItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
}

// Deletes an account-owned item category. Default system categories cannot be deleted.
type DeleteItemCategoryEndpoint struct{}

func (e *DeleteItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteItemCategoryRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteItemCategoryRequest, *apiresource.EmptyResource]{
		Title:             "Delete Item Category",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/item-categories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteItemCategoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).DeleteItemCategory
		},
	})
}
