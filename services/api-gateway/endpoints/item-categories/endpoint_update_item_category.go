package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateItemCategoryRequest is the request to partially update an item category.
type UpdateItemCategoryRequest struct {
	// The ID of the item category to update.
	ItemCategoryID string `path:"id" validate:"required"`
	// The display name of the item category.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Optional notes about the item category.
	Notes *string `json:"notes,omitempty" nullable:"false"`
}

var sampleUpdateItemCategoryRequest = &UpdateItemCategoryRequest{
	Name: new("Electronic Components"),
}

func (*UpdateItemCategoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemCategoryRequest)
}

type UpdateItemCategoryEndpoint struct{}

func (e *UpdateItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemCategoryRequest, *apiresource.ItemCategory] {
	return &apiendpoint.APIEndpoint[*UpdateItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Update Item Category",
		Description:       "Partially updates an account-owned item category. Default system categories cannot be updated.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/item-categories/{id}",
		ContentType:       "application/json",
		Request:           &UpdateItemCategoryRequest{},
		Response:          &apiresource.ItemCategory{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).UpdateItemCategory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group"},
		}),
	}
}
