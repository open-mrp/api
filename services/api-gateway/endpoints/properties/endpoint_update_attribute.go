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
	"github.com/open-mrp/apikit/field"
)

// Request to update an attribute.
type UpdateAttributeRequest struct {
	// The property the attribute belongs to.
	PropertyID string `path:"property_id" validate:"required"`
	// Attribute ID.
	AttributeID string `path:"id" validate:"required"`
	// The value, such as `Large`.
	//
	// Unique (case-insensitive) within its property. A new value shows everywhere the attribute is assigned.
	Value field.Optional[string] `json:"value,omitzero" validate:"omitempty,max=255"`
	// Swatch color shown for the attribute.
	ColorCode field.Optional[constants.Color] `json:"color,omitzero"`
	// Position within the property, starting at `1`.
	//
	// At most the property's attribute count; the attributes between the old and new positions shift to make room.
	SortOrder field.Optional[int32] `json:"sort_order,omitzero" validate:"omitempty,min=1"`
	// `active` or `archived`. Archived attributes stay on the items that have them but can't be newly assigned.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleUpdateAttributeRequest = &UpdateAttributeRequest{
	Value:     field.Some("Blue"),
	ColorCode: field.Some(constants.ColorBlue),
	SortOrder: field.Some(int32(2)),
}

func (*UpdateAttributeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAttributeRequest)
}

// Updates an attribute.
//
// Only fields you send are changed. A new `value` shows everywhere the attribute is assigned.
type UpdateAttributeEndpoint struct{}

func (e *UpdateAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAttributeRequest, *apiresource.Attribute] {
	return (&apiendpoint.APIEndpoint[*UpdateAttributeRequest, *apiresource.Attribute]{
		Title:             "Update Attribute",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             CatalogPropertyAttributeRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeAttribute,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAttribute,
			Fields:     []string{"property"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).UpdateAttribute
		},
	})
}
