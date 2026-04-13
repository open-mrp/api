package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListItemCategoriesRequest is the request to list item categories with optional filters.
type ListItemCategoriesRequest struct {
	apiresource.PaginationRequest
	// Filter by item category type code (material_category or product_category).
	Type *constants.ItemCategoryType `query:"type"`
}

type ListItemCategoriesEndpoint struct{}

func (e *ListItemCategoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListItemCategoriesRequest, *apiresource.List[apiresource.ItemCategory]] {
	return &apiendpoint.APIEndpoint[*ListItemCategoriesRequest, *apiresource.List[apiresource.ItemCategory]]{
		Title:             "List Item Categories",
		Description:       "Returns a paginated list of item categories for the current account, including account-specific and global system categories.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/item-categories",
		Request:           &ListItemCategoriesRequest{},
		Response:          &apiresource.List[apiresource.ItemCategory]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemCategoriesRequest) (*apiresource.List[apiresource.ItemCategory], *apierror.APIError) {
			return svc.(ItemCategorySvc).ListItemCategories
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group"},
		}),
	}
}
