package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetAccountGroupProductLineAccessRequest is a request to get product line access for an account group.
type GetAccountGroupProductLineAccessRequest struct {
	// Account group ID.
	AccountGroupID string `path:"account_group_id" validate:"required"`
}

type GetAccountGroupProductLineAccessEndpoint struct{}

func (e *GetAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess] {
	return &apiendpoint.APIEndpoint[*GetAccountGroupProductLineAccessRequest, *apiresource.AccountGroupProductLineAccess]{
		Title:             "Get Account Group Product Line Access",
		Description:       "Returns product line access for an account group.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
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
