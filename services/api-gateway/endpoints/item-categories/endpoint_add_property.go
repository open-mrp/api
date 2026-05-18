package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to add a property to an item category.
type AddItemCategoryPropertyRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// Property ID.
	PropertyID string `path:"property_id" validate:"required"`
}

// Adds a property to an item category. Default system categories cannot be modified.
type AddItemCategoryPropertyEndpoint struct{}

func (e *AddItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddItemCategoryPropertyRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*AddItemCategoryPropertyRequest, *apiresource.EmptyResource]{
		Title:             "Add Item Category Property",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/item-categories/{id}/properties/{property_id}",
		ContentType:       "application/json",
		Request:           &AddItemCategoryPropertyRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).AddItemCategoryProperty
		},
	}).WithDocSource(e)
}
