package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ChangeItemCategoryRequest is the request to change an item's category.
type ChangeItemCategoryRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// ID of the category to move the item to.
	//
	// The category's type must be compatible with the item's type; otherwise the request fails validation.
	CategoryID string `path:"category_id" validate:"required"`
}

// Moves an item to a different category.
//
// The item's rate units (unit value, unit cost, burn rate) and any related order-point, consumption, and production quantity units are updated to the new category's base unit. Re-assigning the item's current category is a no-op.
type ChangeItemCategoryEndpoint struct{}

func (e *ChangeItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeItemCategoryRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*ChangeItemCategoryRequest, *apiresource.Item]{
		Title:             "Change Item Category",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/category/{category_id}",
		SDKMethodKey:      "change_category",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeItemCategoryRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).ChangeItemCategory
		},
		ObjectType: constants.ObjectTypeItem,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes", "category.unit_group", "category.properties", "category.unit_group.base_unit", "category.unit_group.associated_units", "category.unit_group.associated_units.unit"},
		}),
	})
}
