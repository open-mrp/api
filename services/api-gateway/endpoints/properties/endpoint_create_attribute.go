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

// CreateAttributeRequest is the request to create a new attribute.
type CreateAttributeRequest struct {
	// The ID of the property to create the attribute under.
	PropertyID string `path:"property_id" validate:"required"`
	// The value of the attribute.
	Value string `json:"value" validate:"required"`
	// The color code of the attribute. Randomly assigned if not provided.
	ColorCode *constants.Color `json:"color,omitempty"`
	// The display order of the attribute. Defaults to last position if not provided.
	SortOrder *int32 `json:"sort_order" validate:"omitempty,min=1"`
}

var (
	sampleCreateAttributeSortOrder int32           = 1
	sampleCreateAttributeColor     constants.Color = constants.ColorRed
)

var sampleCreateAttributeRequest = &CreateAttributeRequest{
	Value:     "Red",
	ColorCode: &sampleCreateAttributeColor,
	SortOrder: &sampleCreateAttributeSortOrder,
}

func (*CreateAttributeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAttributeRequest)
}

type CreateAttributeEndpoint struct{}

func (e *CreateAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAttributeRequest, *apiresource.Attribute] {
	return &apiendpoint.APIEndpoint[*CreateAttributeRequest, *apiresource.Attribute]{
		Title:             "Create Attribute",
		Description:       "Creates a new attribute under a property.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties/{property_id}/attributes",
		Request:           &CreateAttributeRequest{},
		Response:          &apiresource.Attribute{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).CreateAttribute
		},
	}
}
