package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to change an item's category.
type ChangeItemCategoryRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// ID of the category to move the item to.
	//
	// The category's type has to suit the item: a material can only move to a material category, and a product or part can only move to a product category. Anything else fails validation.
	//
	// The category also has to carry the properties of every attribute the item already has, since the move keeps those attributes. Unlink the offending attributes, or add their properties to the target category, before moving the item.
	CategoryID string `path:"category_id" validate:"required"`
}

// Moves an item to a different category and returns the updated item.
//
// The item's rate units (unit value, unit cost, burn rate) and any related order-point, consumption, and production quantity units are switched to the new category's base unit. Only the units change — the numbers attached to them are carried over as they were, so review any figure whose meaning depends on the unit after moving between categories that count differently.
//
// Re-assigning the item's current category succeeds and changes nothing.
type ChangeItemCategoryEndpoint struct{}

func (e *ChangeItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeItemCategoryRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*ChangeItemCategoryRequest, *apiresource.Item]{
		Title:               "Change Item Category",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/category/{category_id}",
		SDKMethodKey:        "change_category",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}},
		Preview:             true,
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
