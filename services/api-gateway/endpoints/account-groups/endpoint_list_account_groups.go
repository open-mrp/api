package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list account groups.
type ListAccountGroupsRequest struct {
	apiresource.PaginationRequest
	// Account group type filter.
	Type *constants.AccountGroupType `query:"type"`
}

// Returns a paginated list of account groups.
type ListAccountGroupsEndpoint struct{}

func (e *ListAccountGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountGroupsRequest, *apiresource.List[apiresource.AccountGroup]] {
	return (&apiendpoint.APIEndpoint[*ListAccountGroupsRequest, *apiresource.List[apiresource.AccountGroup]]{
		Title:             "List Account Groups",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/account-groups",
		Request:           &ListAccountGroupsRequest{},
		Response:          &apiresource.List[apiresource.AccountGroup]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountGroupsRequest) (*apiresource.List[apiresource.AccountGroup], *apierror.APIError) {
			return svc.(AccountGroupSvc).ListAccountGroups
		},
	}).WithDocSource(e)
}
