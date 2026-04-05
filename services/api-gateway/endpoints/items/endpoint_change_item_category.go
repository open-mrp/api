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
	// The ID of the item.
	ItemID string `path:"id" validate:"required"`
	// The ID of the new category.
	CategoryID string `path:"category_id" validate:"required"`
}

type ChangeItemCategoryEndpoint struct{}

func (e *ChangeItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeItemCategoryRequest, *apiresource.Item] {
	return &apiendpoint.APIEndpoint[*ChangeItemCategoryRequest, *apiresource.Item]{
		Title:             "Change Item Category",
		Description:       "Changes the category of an item. When the category changes, the item's rate units are updated to the new category's base unit.",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/items/{id}/category/{category_id}",
		Request:           &ChangeItemCategoryRequest{},
		Response:          &apiresource.Item{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeItemCategoryRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).ChangeItemCategory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate"},
		}),
	}
}
