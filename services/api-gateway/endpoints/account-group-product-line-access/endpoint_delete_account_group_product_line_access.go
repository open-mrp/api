package accountgroupproductlineaccessep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteAccountGroupProductLineAccessEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountGroupProductLineAccessRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountGroupProductLineAccessRequest, *object.Deleted]{
		Title:             "Delete Account Group Product Line Access",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/sales/product-line-access/account-groups/{account_group_id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeAccountGroupProductLineAccess,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductLineAccess, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountGroupProductLineAccessRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(AccountGroupProductLineAccessSvc).DeleteAccountGroupProductLineAccess, constants.ObjectTypeAccountGroupProductLineAccess, "account_group_id")
		},
	})
}
