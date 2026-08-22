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
	"github.com/open-mrp/api/shared/field"
)

// Request to update an attribute.
type UpdateAttributeRequest struct {
	// The property the attribute belongs to.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
	// The selectable value this attribute represents, such as `Red`.
	//
	// Must be non-blank and unique across all attributes in the account, not just within the property.
	Value field.Optional[string] `json:"value,omitzero"`
	// Swatch color used to display this attribute in the UI.
	ColorCode field.Optional[constants.Color] `json:"color,omitzero"`
	// New position of this attribute relative to its siblings within the property, starting at `1`.
	//
	// Must be at most the property's current attribute count; the attributes between the old and new positions shift to make room.
	SortOrder field.Optional[int32] `json:"sort_order,omitzero" validate:"omitempty,min=1"`
}

var sampleUpdateAttributeRequest = &UpdateAttributeRequest{
	Value:     field.Some("Blue"),
	ColorCode: field.Some(constants.ColorBlue),
	SortOrder: field.Some(int32(2)),
}

func (*UpdateAttributeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAttributeRequest)
}

// Partially updates an attribute.
//
// Items reference attributes by ID, so changing the value renames the attribute everywhere it is already assigned.
type UpdateAttributeEndpoint struct{}

func (e *UpdateAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAttributeRequest, *apiresource.Attribute] {
	return (&apiendpoint.APIEndpoint[*UpdateAttributeRequest, *apiresource.Attribute]{
		Title:               "Update Attribute",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               CatalogPropertyAttributeRoute,
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAttribute,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).UpdateAttribute
		},
	})
}
