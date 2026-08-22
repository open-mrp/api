package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list system properties.
type ListSysPropertiesRequest struct {
	apiresource.PaginationRequest
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
		Preview:           true,
		ObjectType:        constants.ObjectTypeSysProperty,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSystemProperties, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSysPropertiesRequest) (*apiresource.List[apiresource.SysProperty], *apierror.APIError) {
			return svc.(SysPropertySvc).ListSysProperties
		},
	})
}
