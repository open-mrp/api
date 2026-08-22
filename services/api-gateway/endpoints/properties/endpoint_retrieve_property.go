package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a property.
type RetrievePropertyRequest struct {
	// Property ID.
	PropertyID string `path:"id" validate:"required"`
}

// Returns a property by ID.
type RetrievePropertyEndpoint struct{}

func (e *RetrievePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePropertyRequest, *apiresource.Property] {
	return (&apiendpoint.APIEndpoint[*RetrievePropertyRequest, *apiresource.Property]{
		Title:               "Retrieve Property",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               CatalogPropertyRoute,
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeProperty,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePropertyRequest) (*apiresource.Property, *apierror.APIError) {
			return svc.(PropertySvc).GetProperty
		},
	})
}
