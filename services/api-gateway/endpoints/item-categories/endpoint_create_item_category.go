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

// CreateItemCategoryRequest is the request to create a new item category.
type CreateItemCategoryRequest struct {
	// The display name of the item category.
	Name string `json:"name" validate:"required"`
	// The type of item category (material_category or product_category).
	Type string `json:"type" validate:"required"`
	// The ID of the unit group to associate with this item category.
	UnitGroupID string `json:"unit_group_id" validate:"required"`
}

var sampleCreateItemCategoryRequest = &CreateItemCategoryRequest{
	Name:        "Electronics",
	Type:        string(constants.ItemCategoryTypeMaterial),
	UnitGroupID: apiresource.SampleUnitGroupID,
}

func (*CreateItemCategoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateItemCategoryRequest)
}

type CreateItemCategoryEndpoint struct{}

func (e *CreateItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateItemCategoryRequest, *apiresource.ItemCategory] {
	return &apiendpoint.APIEndpoint[*CreateItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Create Item Category",
		Description:       "Creates a new account-owned item category.",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/item-categories",
		Request:           &CreateItemCategoryRequest{},
		Response:          &apiresource.ItemCategory{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).CreateItemCategory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "properties", "unit_group"},
		}),
	}
}
