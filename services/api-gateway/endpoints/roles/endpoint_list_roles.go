package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListRolesRequest is a request to list roles.
type ListRolesRequest struct {
	apiresource.PaginationRequest
	// Filter results to roles whose type matches any of the given values.
	RoleType []constants.RoleType `query:"types"`
}

// Returns a paginated list of roles for the target account, including global roles.
type ListRolesEndpoint struct{}

func (e *ListRolesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRolesRequest, *apiresource.List[apiresource.Role]] {
	return (&apiendpoint.APIEndpoint[*ListRolesRequest, *apiresource.List[apiresource.Role]]{
		Title:             "List Roles",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/roles",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
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
