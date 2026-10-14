package itemcategoryep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve an item category.
type RetrieveItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
}

// Retrieves an item category.
//
// Works for your account's categories and for system categories.
type RetrieveItemCategoryEndpoint struct{}

func (e *RetrieveItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemCategoryRequest, *apiresource.ItemCategory] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Retrieve Item Category",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/item-categories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).GetItemCategory
		},
		IncludeConfig: itemCategoryIncludes(),
	})
}
