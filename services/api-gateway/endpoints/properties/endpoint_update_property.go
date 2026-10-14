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

// Request to update a property.
type UpdatePropertyRequest struct {
	// Property ID.
	PropertyID string `path:"id" validate:"required"`
	// Display name, such as `Size`.
	//
	// Unique (case-insensitive) among your account's properties.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdatePropertyRequest = &UpdatePropertyRequest{
	Name: field.Some("Size"),
}

func (*UpdatePropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePropertyRequest)
}

// Updates a property.
//
// Only fields you send are changed.
type UpdatePropertyEndpoint struct{}

func (e *UpdatePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePropertyRequest, *apiresource.Property] {
	return (&apiendpoint.APIEndpoint[*UpdatePropertyRequest, *apiresource.Property]{
		Title:             "Update Property",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             CatalogPropertyRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeProperty,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePropertyRequest) (*apiresource.Property, *apierror.APIError) {
			return svc.(PropertySvc).UpdateProperty
		},
	})
}
