package itemcategoryep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to add a property to an item category.
type AddItemCategoryPropertyRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// ID of the property to add to the category.
	PropertyID string `path:"property_id" validate:"required"`
}

// Attaches one of your account's properties to an item category.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge sets `property_ids` on Create or Update Item Category.
type AddItemCategoryPropertyEndpoint struct{}

func (e *AddItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddItemCategoryPropertyRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*AddItemCategoryPropertyRequest, *apiresource.EmptyResource]{
		Title:             "Add Item Category Property",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/item-categories/{id}/properties/{property_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddItemCategoryPropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).AddItemCategoryProperty
		},
	})
}
