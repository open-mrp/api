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

// Request to create an item category.
type CreateItemCategoryRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Item category type (material_category or product_category).
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// Unit group ID.
	UnitGroupID string `json:"unit_group_id" validate:"required,max=191"`
}

var sampleCreateItemCategoryRequest = &CreateItemCategoryRequest{
	Name:        "Electronics",
	Type:        constants.ItemCategoryTypeMaterial,
	UnitGroupID: apiresource.SampleUnitGroupID,
}

func (*CreateItemCategoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateItemCategoryRequest)
}

type CreateItemCategoryEndpoint struct{}

func (e *CreateItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateItemCategoryRequest, *apiresource.ItemCategory] {
	return &apiendpoint.APIEndpoint[*CreateItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Create Item Category",
		Description:       "Creates an account-owned item category.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/item-categories",
		Request:           &CreateItemCategoryRequest{},
		Response:          &apiresource.ItemCategory{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).CreateItemCategory
		},
		LocationFunc: func(resp *apiresource.ItemCategory) string {
			return "/v1/catalog/item-categories/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group"},
		}),
	}
}
