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

// Request to list item categories.
type ListItemCategoriesRequest struct {
	apiresource.PaginationRequest
	// Filter by item category type.
	Type *constants.ItemCategoryType `query:"type"`
}

// Returns a paginated list of the item categories available to the current account, newest first.
//
// Both the account's own categories and the platform-provided system categories are included. The `q` search term is matched against the category name.
type ListItemCategoriesEndpoint struct{}

func (e *ListItemCategoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListItemCategoriesRequest, *apiresource.List[apiresource.ItemCategory]] {
	return (&apiendpoint.APIEndpoint[*ListItemCategoriesRequest, *apiresource.List[apiresource.ItemCategory]]{
		Title:               "List Item Categories",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/item-categories",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemCategoriesRequest) (*apiresource.List[apiresource.ItemCategory], *apierror.APIError) {
			return svc.(ItemCategorySvc).ListItemCategories
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group", "unit_group.base_unit", "unit_group.associated_units", "unit_group.associated_units.unit"},
		}),
	})
}
