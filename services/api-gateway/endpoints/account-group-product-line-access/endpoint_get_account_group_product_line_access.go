package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetAccountGroupProductLineAccessRequest is the request to retrieve product line access for a single account group.
type GetAccountGroupProductLineAccessRequest struct {
	// The ID of the account group.
	AccountGroupID string `path:"account_group_id" validate:"required"`
}

type GetAccountGroupProductLineAccessEndpoint struct{}

func (e *GetAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess] {
	return &apiendpoint.APIEndpoint[*GetAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess]{
		Title:             "Get Account Group Product Line Access",
		Description:       "Returns the product line access for a single account group.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		Request:           &GetAccountGroupProductLineAccessRequest{},
		Response:          &apiresource.AccountGroupProductLineAccess{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).GetAccountGroupProductLineAccess
		},
	}
}
