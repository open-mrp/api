package propertyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list properties.
type ListPropertiesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of properties for the target account.
//
// Properties come back newest first. The `q` search term is matched against the property name.
type ListPropertiesEndpoint struct{}

func (e *ListPropertiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPropertiesRequest, *apiresource.List[apiresource.Property]] {
	return (&apiendpoint.APIEndpoint[*ListPropertiesRequest, *apiresource.List[apiresource.Property]]{
		Title:             "List Properties",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             CatalogPropertiesRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeProperty,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPropertiesRequest) (*apiresource.List[apiresource.Property], *apierror.APIError) {
			return svc.(PropertySvc).ListProperties
		},
	})
}
