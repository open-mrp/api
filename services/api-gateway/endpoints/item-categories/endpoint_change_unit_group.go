package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ChangeItemCategoryUnitGroupRequest is the request to change the unit group of an item category.
type ChangeItemCategoryUnitGroupRequest struct {
	// The ID of the item category.
	ItemCategoryID string `path:"id" validate:"required"`
	// The ID of the new unit group.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
}

type ChangeItemCategoryUnitGroupEndpoint struct{}

func (e *ChangeItemCategoryUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeItemCategoryUnitGroupRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*ChangeItemCategoryUnitGroupRequest, *apiresource.EmptyResource]{
		Title:             "Change Item Category Unit Group",
		Description:       "Changes the unit group associated with an item category. All items in the category are updated to use the new base unit asynchronously.",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/item-categories/{id}/unit-groups/{unit_group_id}",
		ContentType:       "application/json",
		Request:           &ChangeItemCategoryUnitGroupRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeItemCategoryUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemCategorySvc).ChangeItemCategoryUnitGroup
		},
	}
}
