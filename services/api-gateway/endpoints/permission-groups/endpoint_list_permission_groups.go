package permissiongroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListPermissionGroupsRequest is the request to list permission groups.
type ListPermissionGroupsRequest struct {
	apiresource.PaginationRequest
}

type ListPermissionGroupsEndpoint struct{}

func (e *ListPermissionGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPermissionGroupsRequest, *apiresource.List[apiresource.PermissionGroup]] {
	return &apiendpoint.APIEndpoint[*ListPermissionGroupsRequest, *apiresource.List[apiresource.PermissionGroup]]{
		Title:             "List Permission Groups",
		Description:       "Returns a paginated list of permission groups with their nested permissions.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/permission-groups",
		Request:           &ListPermissionGroupsRequest{},
		Response:          &apiresource.List[apiresource.PermissionGroup]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPermissionGroupsRequest) (*apiresource.List[apiresource.PermissionGroup], *apierror.APIError) {
			return svc.(PermissionGroupSvc).ListPermissionGroups
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePermissionGroup,
			Fields:     []string{"owner"},
		}),
	}
}
