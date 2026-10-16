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
	// Matches roles whose `name` has a word starting with the term.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return roles with any of these types.
	RoleType []constants.RoleType `query:"types"`
}

// Returns a list of the roles you can assign in your account.
//
// Sorted by creation date, newest first. Includes your account's roles and the system roles OpenMRP shares with every account.
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
