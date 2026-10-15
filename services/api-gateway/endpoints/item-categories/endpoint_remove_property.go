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

// Request to remove a property from an item category.
type RemoveItemCategoryPropertyRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// ID of the property to remove from the category.
	PropertyID string `path:"property_id" validate:"required"`
}

// Detaches a property from an item category.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge sets `property_ids` on Create or Update Item Category.
type RemoveItemCategoryPropertyEndpoint struct{}

func (e *RemoveItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveItemCategoryPropertyRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*RemoveItemCategoryPropertyRequest, *object.Deleted]{
		Title:             "Remove Item Category Property",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/item-categories/{id}/properties/{property_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeProperty,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveItemCategoryPropertyRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ItemCategorySvc).RemoveItemCategoryProperty, constants.ObjectTypeProperty, "property_id")
		},
	})
}
