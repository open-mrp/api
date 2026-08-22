package propertyep

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

// Request to create a property.
type CreatePropertyRequest struct {
	// Display name of the property, such as `Color` or `Size`.
	//
	// Must be unique within your account.
	Name string `json:"name" validate:"required,max=255"`
}

var sampleCreatePropertyRequest = &CreatePropertyRequest{
	Name: "Color",
}

func (*CreatePropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePropertyRequest)
}

// Creates a property.
//
// The property starts with no attributes; add its selectable values afterwards with the create attribute endpoint. Returns a conflict error if a property with the same name already exists.
type CreatePropertyEndpoint struct{}

func (e *CreatePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePropertyRequest, *apiresource.Property] {
	return (&apiendpoint.APIEndpoint[*CreatePropertyRequest, *apiresource.Property]{
		Title:               "Create Property",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               CatalogPropertiesRoute,
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeProperty,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePropertyRequest) (*apiresource.Property, *apierror.APIError) {
			return svc.(PropertySvc).CreateProperty
		},
		LocationFunc: func(resp *apiresource.Property) string {
			return "/v1/catalog/properties/" + resp.ID
		},
	})
}
