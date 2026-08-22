package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an item category.
type RetrieveItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
}

// Returns an item category by ID.
//
// Both account-owned categories and global system categories can be retrieved.
type RetrieveItemCategoryEndpoint struct{}

func (e *RetrieveItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemCategoryRequest, *apiresource.ItemCategory] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemCategoryRequest, *apiresource.ItemCategory]{
		Title:               "Retrieve Item Category",
		Method:              http.MethodGet,
		Route:               "/v1/catalog/item-categories/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).GetItemCategory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group", "unit_group.base_unit", "unit_group.associated_units", "unit_group.associated_units.unit"},
		}),
	})
}
