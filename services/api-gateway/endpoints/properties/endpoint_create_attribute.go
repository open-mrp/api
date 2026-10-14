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

// Request to create an attribute.
type CreateAttributeRequest struct {
	// The property to add this attribute to.
	PropertyID string `path:"property_id" validate:"required"`
	// The value, such as `Large`.
	//
	// Unique (case-insensitive) within its property. Leading and trailing whitespace is trimmed.
	Value string `json:"value" validate:"required,max=255"`
	// Swatch color shown for the attribute. Defaults to `default`, a neutral swatch.
	ColorCode field.Optional[constants.Color] `json:"color,omitzero"`
	// Position within the property, starting at `1`. Defaults to last.
	//
	// At most the property's attribute count plus one; attributes at or after it shift down by one.
	SortOrder field.Optional[int32] `json:"sort_order,omitzero" validate:"omitempty,min=1"`
	// `active` or `archived`. Defaults to `active`. Archived attributes can't be newly assigned to items.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
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

// Adds an attribute to a property.
//
// Attributes after the chosen `sort_order` shift down by one.
type CreateAttributeEndpoint struct{}

func (e *CreateAttributeEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAttributeRequest, *apiresource.Attribute] {
	return (&apiendpoint.APIEndpoint[*CreateAttributeRequest, *apiresource.Attribute]{
		Title:             "Create Attribute",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             CatalogPropertyAttributesRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeAttribute,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAttribute,
			Fields:     []string{"property"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAttributeRequest) (*apiresource.Attribute, *apierror.APIError) {
			return svc.(PropertySvc).CreateAttribute
		},
	})
}
