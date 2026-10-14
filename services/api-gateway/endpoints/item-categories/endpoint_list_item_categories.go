package itemcategoryep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list item categories.
type ListItemCategoriesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Filter by item category type.
	Type *constants.ItemCategoryType `query:"type"`
}

// Returns a paginated list of the item categories available to the current account, newest first.
//
// Both the account's own categories and the platform-provided system categories are included. The `q` search term is matched against the category name.
type ListItemCategoriesEndpoint struct{}

func (e *ListItemCategoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListItemCategoriesRequest, *apiresource.List[apiresource.ItemCategory]] {
	return (&apiendpoint.APIEndpoint[*ListItemCategoriesRequest, *apiresource.List[apiresource.ItemCategory]]{
		Title:             "List Item Categories",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/item-categories",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemCategoriesRequest) (*apiresource.List[apiresource.ItemCategory], *apierror.APIError) {
			return svc.(ItemCategorySvc).ListItemCategories
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group", "unit_group.base_unit", "unit_group.associated_units", "unit_group.associated_units.unit"},
		}),
	})
}
