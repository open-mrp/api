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

// Request to partially update an item category.
type UpdateItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Notes.
	Notes *string `json:"notes,omitempty"`
}

var sampleUpdateItemCategoryRequest = &UpdateItemCategoryRequest{
	Name: new("Electronic Components"),
}

func (*UpdateItemCategoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemCategoryRequest)
}

// Partially updates an account-owned item category. Default system categories cannot be updated.
type UpdateItemCategoryEndpoint struct{}

func (e *UpdateItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemCategoryRequest, *apiresource.ItemCategory] {
	return (&apiendpoint.APIEndpoint[*UpdateItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Update Item Category",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/item-categories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).UpdateItemCategory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group", "unit_group.base_unit", "unit_group.associated_units", "unit_group.associated_units.unit"},
		}),
	})
}
