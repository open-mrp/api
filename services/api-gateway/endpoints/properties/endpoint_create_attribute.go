package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create an attribute.
type CreateAttributeRequest struct {
	// Property ID.
	PropertyID string `path:"property_id" validate:"required"`
	// The selectable value this attribute represents, such as `Red`.
	//
	// Must be unique across all attributes in the account, not just within the property. Leading and trailing whitespace is trimmed.
	Value string `json:"value" validate:"required"`
	// Swatch color used to display this attribute in the UI.
	//
	// When omitted, one of the nine named colors (everything except `default`) is assigned at random.
	ColorCode field.Optional[constants.Color] `json:"color,omitzero"`
	// Position of the new attribute relative to its siblings within the property, starting at `1`.
	//
	// Must be at most the property's current attribute count plus one; siblings at or after this position are shifted one position later. Defaults to the last position if not provided.
	SortOrder field.Optional[int32] `json:"sort_order,omitzero" validate:"omitempty,min=1"`
}

var (
	sampleCreateAttributeSortOrder int32           = 1
	sampleCreateAttributeColor     constants.Color = constants.ColorRed
)

var sampleCreateAttributeRequest = &CreateAttributeRequest{
	Value:     "Red",
	ColorCode: field.Some(sampleCreateAttributeColor),
	SortOrder: field.Some(sampleCreateAttributeSortOrder),
}

func (*CreateAttributeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAttributeRequest)
}

// Creates an attribute under a property.
type CreateAttributeEndpoint struct{}

func (e *CreateAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAttributeRequest, *apiresource.Attribute] {
	return (&apiendpoint.APIEndpoint[*CreateAttributeRequest, *apiresource.Attribute]{
		Title:               "Create Attribute",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               CatalogPropertyAttributesRoute,
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAttribute,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).CreateAttribute
		},
	})
}
