package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// RemoveItemCategoryPropertyRequest is the request to remove a property from an item category.
type RemoveItemCategoryPropertyRequest struct {
	// The ID of the item category.
	ItemCategoryID string `path:"id" validate:"required"`
	// The ID of the property to remove.
	PropertyID string `path:"property_id" validate:"required"`
}

type RemoveItemCategoryPropertyEndpoint struct{}

func (e *RemoveItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveItemCategoryPropertyRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RemoveItemCategoryPropertyRequest, *apiresource.EmptyResource]{
		Title:             "Remove Item Category Property",
		Description:       "Removes a property from an item category. Default system categories cannot be modified.",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/item-categories/{id}/properties/{property_id}",
		ContentType:       "application/json",
		Request:           &RemoveItemCategoryPropertyRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).RemoveItemCategoryProperty
		},
	}
}
