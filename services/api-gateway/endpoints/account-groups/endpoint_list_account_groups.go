package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list account groups.
type ListAccountGroupsRequest struct {
	apiresource.PaginationRequest
	// Filters results to account groups of the given type.
	Type *constants.AccountGroupType `query:"type"`
}

// Returns a paginated list of account groups, newest first.
//
// The `q` search term matches the group's name and description.
type ListAccountGroupsEndpoint struct{}

func (e *ListAccountGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountGroupsRequest, *apiresource.List[apiresource.AccountGroup]] {
	return (&apiendpoint.APIEndpoint[*ListAccountGroupsRequest, *apiresource.List[apiresource.AccountGroup]]{
		Title:               "List Account Groups",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/sales/account-groups",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAccountGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountGroupsRequest) (*apiresource.List[apiresource.AccountGroup], *apierror.APIError) {
			return svc.(AccountGroupSvc).ListAccountGroups
		},
	})
}
