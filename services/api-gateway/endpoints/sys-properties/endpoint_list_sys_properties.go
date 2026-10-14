package syspropertyep

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

// Request to list system properties.
type ListSysPropertiesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of system properties for the current account.
//
// A counter appears here only once its number series has been used at least once, so an account may have fewer counters than there are counter types. The `q` search term is matched against the counter type name.
type ListSysPropertiesEndpoint struct{}

func (e *ListSysPropertiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSysPropertiesRequest, *apiresource.List[apiresource.SysProperty]] {
	return (&apiendpoint.APIEndpoint[*ListSysPropertiesRequest, *apiresource.List[apiresource.SysProperty]]{
		Title:             "List System Properties",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/properties",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSysProperty,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSystemProperties, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSysPropertiesRequest) (*apiresource.List[apiresource.SysProperty], *apierror.APIError) {
			return svc.(SysPropertySvc).ListSysProperties
		},
	})
}
