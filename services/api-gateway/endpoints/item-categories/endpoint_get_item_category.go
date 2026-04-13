package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an item category.
type GetItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
}

type GetItemCategoryEndpoint struct{}

func (e *GetItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemCategoryRequest, *apiresource.ItemCategory] {
	return &apiendpoint.APIEndpoint[*GetItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Get Item Category",
		Description:       "Returns an item category by ID. Includes account-specific and global system categories.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/item-categories/{id}",
		ContentType:       "application/json",
		Request:           &GetItemCategoryRequest{},
		Response:          &apiresource.ItemCategory{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).GetItemCategory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group"},
		}),
	}
}
