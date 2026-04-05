package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteAccountGroupProductLineAccessRequest is the request to delete all product line access for an account group.
type DeleteAccountGroupProductLineAccessRequest struct {
	// The ID of the account group.
	AccountGroupID string `path:"account_group_id" validate:"required"`
}

type DeleteAccountGroupProductLineAccessEndpoint struct{}

func (e *DeleteAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountGroupProductLineAccessRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteAccountGroupProductLineAccessRequest, *apiresource.EmptyResource]{
		Title:             "Delete Account Group Product Line Access",
		Description:       "Removes all product line access for an account group.",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		Request:           &DeleteAccountGroupProductLineAccessRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountGroupProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).DeleteAccountGroupProductLineAccess
		},
	}
}
