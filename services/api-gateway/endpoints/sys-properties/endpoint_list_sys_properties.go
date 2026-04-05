package syspropertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListSysPropertiesRequest is the request to list system properties.
type ListSysPropertiesRequest struct {
	apiresource.PaginationRequest
}

type ListSysPropertiesEndpoint struct{}

func (e *ListSysPropertiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSysPropertiesRequest, *apiresource.List[apiresource.SysProperty]] {
	return &apiendpoint.APIEndpoint[*ListSysPropertiesRequest, *apiresource.List[apiresource.SysProperty]]{
		Title:             "List System Properties",
		Description:       "Returns a paginated list of system properties for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/core/sys-properties",
		Request:           &ListSysPropertiesRequest{},
		Response:          &apiresource.List[apiresource.SysProperty]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSysPropertiesRequest) (*apiresource.List[apiresource.SysProperty], *apierror.APIError) {
			return svc.(SysPropertySvc).ListSysProperties
		},
	}
}
