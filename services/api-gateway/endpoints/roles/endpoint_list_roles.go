package roleep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListRolesRequest is the request to list roles.
type ListRolesRequest struct {
	apiresource.PaginationRequest
	// Filter by role type code(s).
	RoleType []string `query:"role_type"`
}

type ListRolesEndpoint struct{}

func (e *ListRolesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRolesRequest, *apiresource.List[apiresource.Role]] {
	return &apiendpoint.APIEndpoint[*ListRolesRequest, *apiresource.List[apiresource.Role]]{
		Title:             "List Roles",
		Description:       "Returns a paginated list of roles for the target account, including global roles.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/roles",
		Request:           &ListRolesRequest{},
		Response:          &apiresource.List[apiresource.Role]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRolesRequest) (*apiresource.List[apiresource.Role], *apierror.APIError) {
			return svc.(RoleSvc).ListRoles
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRole,
			Fields:     []string{"owner", "permissions"},
		}),
	}
}
