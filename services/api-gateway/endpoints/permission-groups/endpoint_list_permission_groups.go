package permissiongroupep

import (
	"context"
	"net/http"

	"github.com/augno/api/services/auth-service/pkg/types"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list permission groups.
type ListPermissionGroupsRequest struct {
	apiresource.PaginationRequest
}

// Lists the permission catalog, organized into groups of related permissions.
//
// Each group carries the individual permissions it covers; pair a permission's code with an action (`create`, `read`, `update`, or `delete`) to build the permission strings accepted when creating or updating a role. The catalog is platform-defined and identical for every account.
type ListPermissionGroupsEndpoint struct{}

func (e *ListPermissionGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPermissionGroupsRequest, *apiresource.List[apiresource.PermissionGroup]] {
	return (&apiendpoint.APIEndpoint[*ListPermissionGroupsRequest, *apiresource.List[apiresource.PermissionGroup]]{
		Title:             "List Permission Groups",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/permission-groups",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPermissions, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypePermissionGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPermissionGroupsRequest) (*apiresource.List[apiresource.PermissionGroup], *apierror.APIError) {
			return svc.(PermissionGroupSvc).ListPermissionGroups
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePermissionGroup,
			Fields:     []string{"owner"},
		}),
	})
}
