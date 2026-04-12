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

// UpdatePropertyRequest is the request to update a property.
type UpdatePropertyRequest struct {
	// The ID of the property to update.
	PropertyID string `path:"id" validate:"required"`
	// The new name of the property.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
}

var sampleUpdatePropertyRequest = &UpdatePropertyRequest{
	Name: new("Size"),
}

func (*UpdatePropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePropertyRequest)
}

type UpdatePropertyEndpoint struct{}

func (e *UpdatePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePropertyRequest, *apiresource.Property] {
	return &apiendpoint.APIEndpoint[*UpdatePropertyRequest, *apiresource.Property]{
		Title:             "Update Property",
		Description:       "Partially updates a property.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties/{id}",
		Request:           &UpdatePropertyRequest{},
		Response:          &apiresource.Property{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePropertyRequest) (*apiresource.Property, *apierror.APIError) {
			return svc.(PropertySvc).UpdateProperty
		},
	}
}
