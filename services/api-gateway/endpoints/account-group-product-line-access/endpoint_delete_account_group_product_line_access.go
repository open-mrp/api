package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete all product line access for an account group.
type DeleteAccountGroupProductLineAccessRequest struct {
	// Account group ID.
	AccountGroupID string `path:"account_group_id" validate:"required"`
}

// Removes all product line access for an account group.
type DeleteAccountGroupProductLineAccessEndpoint struct{}

func (e *DeleteAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountGroupProductLineAccessRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountGroupProductLineAccessRequest, *apiresource.EmptyResource]{
		Title:             "Delete Account Group Product Line Access",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountGroupProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).DeleteAccountGroupProductLineAccess
		},
	})
}
