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

// UpdateAttributeRequest is the request to update an attribute.
type UpdateAttributeRequest struct {
	// The ID of the property.
	PropertyID string `path:"property_id" validate:"required"`
	// The ID of the attribute to update.
	AttributeID string `path:"id" validate:"required"`
	// The new value of the attribute.
	Value *string `json:"value,omitempty"`
	// The new color code of the attribute.
	ColorCode *constants.Color `json:"color_code,omitempty"`
	// The new display order of the attribute.
	SortOrder *int32 `json:"sort_order,omitempty" validate:"omitempty,min=1"`
}

var sampleUpdateAttributeRequest = &UpdateAttributeRequest{
	Value: new("Blue"),
}

func (*UpdateAttributeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAttributeRequest)
}

type UpdateAttributeEndpoint struct{}

func (e *UpdateAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAttributeRequest, *apiresource.Attribute] {
	return &apiendpoint.APIEndpoint[*UpdateAttributeRequest, *apiresource.Attribute]{
		Title:             "Update Attribute",
		Description:       "Partially updates an attribute.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/properties/{property_id}/attributes/{id}",
		Request:           &UpdateAttributeRequest{},
		Response:          &apiresource.Attribute{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).UpdateAttribute
		},
	}
}
