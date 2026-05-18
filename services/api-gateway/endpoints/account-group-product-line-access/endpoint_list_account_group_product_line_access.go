package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListAccountGroupProductLineAccessRequest is a request to list product line access records grouped by account group.
type ListAccountGroupProductLineAccessRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of product line access records grouped by account group.
type ListAccountGroupProductLineAccessEndpoint struct{}

func (e *ListAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountGroupProductLineAccessRequest, *apiresource.List[apiresource.AccountGroupProductLineAccess]] {
	return (&apiendpoint.APIEndpoint[*ListAccountGroupProductLineAccessRequest, *apiresource.List[apiresource.AccountGroupProductLineAccess]]{
		Title:             "List Account Group Product Line Access",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups",
		Request:           &ListAccountGroupProductLineAccessRequest{},
		Response:          &apiresource.List[apiresource.AccountGroupProductLineAccess]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountGroupProductLineAccessRequest) (*apiresource.List[apiresource.AccountGroupProductLineAccess], *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).ListAccountGroupProductLineAccess
		},
	}).WithDocSource(e)
}
