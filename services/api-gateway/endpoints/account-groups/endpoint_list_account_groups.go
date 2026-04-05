package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListAccountGroupsRequest is the request to list account groups with optional filters.
type ListAccountGroupsRequest struct {
	apiresource.PaginationRequest
	// Filter by account group type code.
	Type *constants.AccountGroupType `query:"type"`
}

type ListAccountGroupsEndpoint struct{}

func (e *ListAccountGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountGroupsRequest, *apiresource.List[apiresource.AccountGroup]] {
	return &apiendpoint.APIEndpoint[*ListAccountGroupsRequest, *apiresource.List[apiresource.AccountGroup]]{
		Title:             "List Account Groups",
		Description:       "Returns a paginated list of account groups.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/account-groups",
		Request:           &ListAccountGroupsRequest{},
		Response:          &apiresource.List[apiresource.AccountGroup]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountGroupsRequest) (*apiresource.List[apiresource.AccountGroup], *apierror.APIError) {
			return svc.(AccountGroupSvc).ListAccountGroups
		},
	}
}
