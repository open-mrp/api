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
)

// Request to create a property on an item category.
type CreateItemCategoryPropertyRequest struct {
	// Item category ID.
	ItemCategoryID string `path:"id" validate:"required"`
	// Display name of the new property, such as `Color` or `Size`.
	//
	// Must be unique within your account. To attach a property that already exists, use the add item category property endpoint.
	Name string `json:"name" validate:"required,max=255"`
}

var sampleCreateItemCategoryPropertyRequest = &CreateItemCategoryPropertyRequest{
	ItemCategoryID: apiresource.SampleItemCategoryID,
	Name:           apiresource.SamplePropertyName,
}

func (*CreateItemCategoryPropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateItemCategoryPropertyRequest)
}

// Creates a property and attaches it to an item category, returning the new property.
//
// The property is one of your account's properties like any other, starting with no attributes, and the category carries it from the moment it exists. Both happen in one request that needs only permission to update the category. A name already used by one of your account's properties returns a conflict error naming `name`.
type CreateItemCategoryPropertyEndpoint struct{}

func (e *CreateItemCategoryPropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateItemCategoryPropertyRequest, *apiresource.Property] {
	return (&apiendpoint.APIEndpoint[*CreateItemCategoryPropertyRequest, *apiresource.Property]{
		Title:               "Create Item Category Property",
		Method:              http.MethodPost,
		Route:               "/v1/catalog/item-categories/{id}/properties",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeProperty,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateItemCategoryPropertyRequest) (*apiresource.Property, *apierror.APIError) {
			return svc.(ItemCategorySvc).CreateItemCategoryProperty
		},
		LocationFunc: func(resp *apiresource.Property) string {
			return "/v1/catalog/properties/" + resp.ID
		},
	})
}
