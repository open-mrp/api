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
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return categories of these types.
	Types []constants.ItemCategoryType `query:"types"`
}

// Returns a list of item categories.
//
// Includes the categories your account created and the system categories every account shares. Categories are sorted by creation date, newest first.
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
		ObjectType:        constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListItemCategoriesRequest) (*apiresource.List[apiresource.ItemCategory], *apierror.APIError) {
			return svc.(ItemCategorySvc).ListItemCategories
		},
		IncludeConfig: itemCategoryIncludes(),
	})
}
