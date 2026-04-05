package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AddItemCategoryPropertyRequest is the request to add a property to an item category.
type AddItemCategoryPropertyRequest struct {
	// The ID of the item category.
	ItemCategoryID string `path:"id" validate:"required"`
	// The ID of the property to add.
	PropertyID string `path:"property_id" validate:"required"`
}

type AddItemCategoryPropertyEndpoint struct{}

func (e *AddItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddItemCategoryPropertyRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*AddItemCategoryPropertyRequest, *apiresource.EmptyResource]{
		Title:             "Add Item Category Property",
		Description:       "Adds a property to an item category. Default system categories cannot be modified.",
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
	}
}
