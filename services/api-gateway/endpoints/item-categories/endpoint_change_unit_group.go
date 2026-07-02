package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to change the unit group of an item category.
type ChangeItemCategoryUnitGroupRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// ID of the unit group to associate with the category.
	//
	// Must have the same unit type as the category's current unit group; otherwise the request fails with a validation error.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
}

// Changes the unit group associated with an item category.
//
// The new unit group must have the same unit type as the current one — for example, a category measured in `mass` units can only switch to another `mass` unit group. Default system categories cannot be modified.
type ChangeItemCategoryUnitGroupEndpoint struct{}

func (e *ChangeItemCategoryUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeItemCategoryUnitGroupRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ChangeItemCategoryUnitGroupRequest, *apiresource.EmptyResource]{
		Title:               "Change Item Category Unit Group",
		Method:              http.MethodPut,
		Route:               "/v1/catalog/item-categories/{id}/unit-groups/{unit_group_id}",
		SDKMethodKey:        "change_unit_group",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeItemCategoryUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).ChangeItemCategoryUnitGroup
		},
	})
}
