package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveAccountGroupProductLineAccessRequest is a request to get product line access for an account group.
type RetrieveAccountGroupProductLineAccessRequest struct {
	// Account group ID.
	AccountGroupID string `path:"account_group_id" validate:"required"`
}

// Returns product line access for an account group.
type RetrieveAccountGroupProductLineAccessEndpoint struct{}

func (e *RetrieveAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess]{
		Title:             "Retrieve Account Group Product Line Access",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).GetAccountGroupProductLineAccess
		},
	})
}
