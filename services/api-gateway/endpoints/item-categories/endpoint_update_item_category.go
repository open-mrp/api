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

// Request to update an item category.
type UpdateItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// Display name, such as `Fasteners`.
	//
	// Unique (case-insensitive) among your account's categories.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Properties the category's items vary by. On update, replaces the whole list.
	//
	// Property names must be unique within the category.
	PropertyIDs field.Optional[[]string] `json:"property_ids,omitzero" validate:"omitempty,max=100"`
	// The lot this category's items are made in, such as one doff or one pallet.
	//
	// The value must be greater than zero, and the unit of the same kind as the category's items. Send `null` to remove the lot convention.
	DefaultLotSize field.Clearable[apirequest.QuantityInput] `json:"default_lot_size,omitzero"`
}

var sampleUpdateItemCategoryRequest = &UpdateItemCategoryRequest{
	Name:        field.Some("Electronic Components"),
	PropertyIDs: field.Some([]string{apiresource.SamplePropertyID}),
}

func (*UpdateItemCategoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemCategoryRequest)
}

// Updates an item category.
//
// Only fields you send are changed. `type` can't change after creation, and system categories can't be updated.
type UpdateItemCategoryEndpoint struct{}

func (e *UpdateItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemCategoryRequest, *apiresource.ItemCategory] {
	return (&apiendpoint.APIEndpoint[*UpdateItemCategoryRequest, *apiresource.ItemCategory]{
		Title:             "Update Item Category",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/item-categories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).UpdateItemCategory
		},
		IncludeConfig: itemCategoryIncludes(),
	})
}
