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

// Request to update a property.
type UpdatePropertyRequest struct {
	// Property ID.
	PropertyID string `path:"id" validate:"required"`
	// Display name of the property, such as `Color` or `Size`.
	//
	// Must be unique within your account.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdatePropertyRequest = &UpdatePropertyRequest{
	Name: field.Some("Size"),
}

func (*UpdatePropertyRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePropertyRequest)
}

// Partially updates a property.
type UpdatePropertyEndpoint struct{}

func (e *UpdatePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePropertyRequest, *apiresource.Property] {
	return (&apiendpoint.APIEndpoint[*UpdatePropertyRequest, *apiresource.Property]{
		Title:               "Update Property",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               CatalogPropertyRoute,
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeProperty,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePropertyRequest) (*apiresource.Property, *apierror.APIError) {
			return svc.(PropertySvc).UpdateProperty
		},
	})
}
