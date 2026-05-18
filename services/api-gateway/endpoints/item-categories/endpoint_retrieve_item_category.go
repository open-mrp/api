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
type RetrieveItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
}

// Returns an item category by ID. Includes account-specific and global system categories.
type RetrieveItemCategoryEndpoint struct{}

func (e *RetrieveItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemCategoryRequest, *apiresource.ItemCategory] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Retrieve Item Category",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/item-categories/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveItemCategoryRequest{},
		Response:          &apiresource.ItemCategory{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).GetItemCategory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group", "unit_group.base_unit", "unit_group.associated_units", "unit_group.associated_units.unit"},
		}),
	}).WithDocSource(e)
}
