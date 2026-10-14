package accountgroupep

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

// Request to list account groups.
type ListAccountGroupsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Filters results to account groups of the given type.
	Type *constants.AccountGroupType `query:"type"`
}

// Returns a paginated list of account groups, newest first.
//
// The `q` search term matches the group's name and description.
type ListAccountGroupsEndpoint struct{}

func (e *ListAccountGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountGroupsRequest, *apiresource.List[apiresource.AccountGroup]] {
	return (&apiendpoint.APIEndpoint[*ListAccountGroupsRequest, *apiresource.List[apiresource.AccountGroup]]{
		Title:             "List Account Groups",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-groups",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountGroupsRequest) (*apiresource.List[apiresource.AccountGroup], *apierror.APIError) {
			return svc.(AccountGroupSvc).ListAccountGroups
		},
	})
}
