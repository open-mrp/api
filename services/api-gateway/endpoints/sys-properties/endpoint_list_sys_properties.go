package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list system properties.
type ListSysPropertiesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of system properties for the current account.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSysPropertiesRequest) (*apiresource.List[apiresource.SysProperty], *apierror.APIError) {
			return svc.(SysPropertySvc).ListSysProperties
		},
	})
}
