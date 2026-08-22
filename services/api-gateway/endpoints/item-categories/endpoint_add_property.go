package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
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
// The property then appears among the category's properties, including in the customer-facing catalog, describing a dimension along which the category's items vary. Each property name can appear only once per category, so attaching a property whose name duplicates one already there returns a conflict error.
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
