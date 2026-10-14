package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to delete all product line access for an account group.
type DeleteAccountGroupProductLineAccessRequest struct {
	// Account group ID.
	AccountGroupID string `path:"account_group_id" validate:"required"`
}

// Removes an account group's product line access record.
//
// Customers in the group keep any product lines granted to them directly or through another of their groups. Removing access from a group that has none returns a not-found error rather than succeeding silently.
type DeleteAccountGroupProductLineAccessEndpoint struct{}

func (e *DeleteAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountGroupProductLineAccessRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountGroupProductLineAccessRequest, *apiresource.EmptyResource]{
		Title:             "Delete Account Group Product Line Access",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountGroupProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountGroupProductLineAccessSvc).DeleteAccountGroupProductLineAccess
		},
	})
}
