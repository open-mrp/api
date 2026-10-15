package propertyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to create a property.
type CreatePropertyRequest struct {
	// Display name, such as `Size`.
	//
	// Unique (case-insensitive) among your account's properties.
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
// It starts with no attributes; add them with Create Attribute.
type CreatePropertyEndpoint struct{}

func (e *CreatePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePropertyRequest, *apiresource.Property] {
	return (&apiendpoint.APIEndpoint[*CreatePropertyRequest, *apiresource.Property]{
		Title:             "Create Property",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             CatalogPropertiesRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeProperty,
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
