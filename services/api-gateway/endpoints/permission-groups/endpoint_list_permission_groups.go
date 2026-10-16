package permissiongroupep

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

// Request to list permission groups.
type ListPermissionGroupsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Lists the permission catalog, organized into groups of related permissions. Internal: 1.0.forge-preview.9 callers
// only; forge reads GET /v1/identity/permissions.
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
		Public:            false,
		AgentTool:         false,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainPermissions, Action: types.ActionRead},
		}},
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
