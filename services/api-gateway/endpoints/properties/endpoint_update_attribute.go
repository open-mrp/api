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

// Request to update an attribute.
type UpdateAttributeRequest struct {
	// Property ID.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
	// Attribute value.
	Value *string `json:"value,omitempty" nullable:"false"`
	// Color code.
	ColorCode *constants.Color `json:"color,omitempty" nullable:"false"`
	// Display order.
	SortOrder *int32 `json:"sort_order,omitempty" nullable:"false" validate:"omitempty,min=1"`
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
		ContentType:       "application/json",
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
