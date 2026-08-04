package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create an item category.
type CreateItemCategoryRequest struct {
	// Display name of the item category.
	Name string `json:"name" validate:"required,max=255"`
	// What kind of items this category groups.
	//
	// - `material_category`: groups raw materials and components (items of type `material`).
	// - `product_category`: groups finished products and parts (items of type `product` or `part`).
	//
	// The type is fixed once the category is created.
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// ID of the unit group that determines the units of measure available to items in this category.
	//
	// Must be one of your account's unit groups or a platform-provided one. After creation the unit group can only be replaced by another unit group of the same unit type, through the Change Item Category Unit Group endpoint.
	UnitGroupID string `json:"unit_group_id" validate:"required"`
}

var sampleCreateItemCategoryRequest = &CreateItemCategoryRequest{
	Name:        "Electronics",
	Type:        constants.ItemCategoryTypeMaterial,
	UnitGroupID: apiresource.SampleUnitGroupID,
}

func (*CreateItemCategoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateItemCategoryRequest)
}

// Creates an item category owned by your account.
//
// The new category starts with no properties; attach them afterwards with the Add Item Category Property endpoint.
type CreateItemCategoryEndpoint struct{}

func (e *CreateItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateItemCategoryRequest, *apiresource.ItemCategory] {
	return (&apiendpoint.APIEndpoint[*CreateItemCategoryRequest, *apiresource.ItemCategory]{
		Title:               "Create Item Category",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/catalog/item-categories",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).CreateItemCategory
		},
		LocationFunc: func(resp *apiresource.ItemCategory) string {
			return "/v1/catalog/item-categories/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group", "unit_group.base_unit", "unit_group.associated_units", "unit_group.associated_units.unit"},
		}),
	})
}
