package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove a property from an item category.
type RemoveItemCategoryPropertyRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// Property ID.
	PropertyID string `path:"property_id" validate:"required"`
}

// Removes a property from an item category. Default system categories cannot be modified.
type RemoveItemCategoryPropertyEndpoint struct{}

func (e *RemoveItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveItemCategoryPropertyRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveItemCategoryPropertyRequest, *apiresource.EmptyResource]{
		Title:             "Remove Item Category Property",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/item-categories/{id}/properties/{property_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).RemoveItemCategoryProperty
		},
	})
}
