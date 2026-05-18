package permissiongroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list permission groups.
type ListPermissionGroupsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of permission groups with their nested permissions.
type ListPermissionGroupsEndpoint struct{}

func (e *ListPermissionGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPermissionGroupsRequest, *apiresource.List[apiresource.PermissionGroup]] {
	return (&apiendpoint.APIEndpoint[*ListPermissionGroupsRequest, *apiresource.List[apiresource.PermissionGroup]]{
		Title:             "List Permission Groups",
		Method:            http.MethodGet,
		ContentType:       "application/json",
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
	}).WithDocSource(e)
}
