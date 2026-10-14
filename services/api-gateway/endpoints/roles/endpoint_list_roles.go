package roleep

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

// Request to list roles.
type ListRolesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Filter results to roles whose type matches any of the given values.
	RoleType []constants.RoleType `query:"types"`
}

// Lists the roles that can be assigned to users in your account, newest first.
//
// Results combine the roles your account owns with the system-owned roles shared by every account. Text search matches the role name.
type ListRolesEndpoint struct{}

func (e *ListRolesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRolesRequest, *apiresource.List[apiresource.Role]] {
	return (&apiendpoint.APIEndpoint[*ListRolesRequest, *apiresource.List[apiresource.Role]]{
		Title:             "List Roles",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainRoles, Action: types.ActionRead}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRolesRequest) (*apiresource.List[apiresource.Role], *apierror.APIError) {
			return svc.(RoleSvc).ListRoles
		},
		ObjectType: constants.ObjectTypeRole,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "owner.account", "permissions"},
		}),
	})
}
