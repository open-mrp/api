package itemcategoryep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to create an item category.
type CreateItemCategoryRequest struct {
	// Display name, such as `Fasteners`.
	//
	// Unique (case-insensitive) among your account's categories.
	Name string `json:"name" validate:"required,max=255"`
	// What kind of items the category holds.
	//
	// `material`: materials. `product`: products and parts. An item can only be in a category of its own kind. Can't change after creation.
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// Properties the category's items vary by.
	//
	// Property names must be unique within the category.
	PropertyIDs []string `json:"property_ids,omitzero" validate:"max=100"`
	// The lot this category's items are made in, such as one doff or one pallet.
	//
	// The value must be greater than zero, and the unit of the same kind as the category's items. Omit it for no lot convention.
	DefaultLotSize field.Optional[apirequest.QuantityInput] `json:"default_lot_size,omitzero"`
	// Unit group the category's items are measured in. Defaults to each.
	UnitGroupID field.Optional[string] `json:"unit_group_id,omitzero" visibility:"internal"`
}

var sampleCreateItemCategoryRequest = &CreateItemCategoryRequest{
	Name:        apiresource.SampleItemCategoryName,
	Type:        constants.ItemCategoryTypeMaterial,
	PropertyIDs: []string{apiresource.SamplePropertyID},
}

func (*CreateItemCategoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateItemCategoryRequest)
}

// Creates an item category.
//
// The new category belongs to your account.
type CreateItemCategoryEndpoint struct{}

func (e *CreateItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateItemCategoryRequest, *apiresource.ItemCategory] {
	return (&apiendpoint.APIEndpoint[*CreateItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Create Item Category",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/item-categories",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).CreateItemCategory
		},
		LocationFunc: func(resp *apiresource.ItemCategory) string {
			return "/v1/catalog/item-categories/" + resp.ID
		},
		IncludeConfig: itemCategoryIncludes(),
	})
}
