package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a property.
type CreatePropertyRequest struct {
	// Name.
	Name string `json:"name" validate:"required,max=255"`
}

var sampleCreatePropertyRequest = &CreatePropertyRequest{
	Name: "Color",
}

func (*CreatePropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePropertyRequest)
}

// Creates a property.
type CreatePropertyEndpoint struct{}

func (e *CreatePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePropertyRequest, *apiresource.Property] {
	return (&apiendpoint.APIEndpoint[*CreatePropertyRequest, *apiresource.Property]{
		Title:             "Create Property",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties",
		Request:           &CreatePropertyRequest{},
		Response:          &apiresource.Property{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
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
	}).WithDocSource(e)
}
