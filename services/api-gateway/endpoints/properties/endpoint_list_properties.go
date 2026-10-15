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
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return properties attached to at least one of these item categories.
	ItemCategoryIDs []string `query:"item_category_ids" validate:"omitempty,max=100"`
}

// Returns a list of properties.
//
// Properties are sorted by creation date, newest first.
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
