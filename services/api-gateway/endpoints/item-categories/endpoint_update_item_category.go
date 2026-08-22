package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update an item category.
type UpdateItemCategoryRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// Display name of the item category.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Free-form notes about the item category.
	Notes field.Optional[string] `json:"notes,omitzero"`
}

var sampleUpdateItemCategoryNotes = "Covers passive and active components; excludes assemblies."
var sampleUpdateItemCategoryRequest = &UpdateItemCategoryRequest{
	Name:  field.Some("Electronic Components"),
	Notes: field.Some(sampleUpdateItemCategoryNotes),
}

func (*UpdateItemCategoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemCategoryRequest)
}

// Updates the name or notes of an item category owned by your account.
//
// Only the fields present in the request body are changed. A category's type is fixed at creation, and its unit group is changed through the Change Item Category Unit Group endpoint. System-owned categories cannot be updated.
type UpdateItemCategoryEndpoint struct{}

func (e *UpdateItemCategoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemCategoryRequest, *apiresource.ItemCategory] {
	return (&apiendpoint.APIEndpoint[*UpdateItemCategoryRequest, *apiresource.ItemCategory]{
		Title:               "Update Item Category",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/item-categories/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeItemCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemCategoryRequest) (*apiresource.ItemCategory, *apierror.APIError) {
			return svc.(ItemCategorySvc).UpdateItemCategory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemCategory,
			Fields:     []string{"owner", "owner.account", "properties", "unit_group", "unit_group.base_unit", "unit_group.associated_units", "unit_group.associated_units.unit"},
		}),
	})
}
