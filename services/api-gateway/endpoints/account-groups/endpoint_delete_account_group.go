package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an account group.
type DeleteAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
}

// Deletes an account group.
//
// Deletion fails with a validation error while the account group is still in use — for example by customer records, product line access, volume discounts, pricing assignments, or an active registration flow.
type DeleteAccountGroupEndpoint struct{}

func (e *DeleteAccountGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountGroupRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountGroupRequest, *apiresource.EmptyResource]{
		Title:               "Delete Account Group",
		Method:              http.MethodDelete,
		Route:               "/v1/sales/account-groups/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomerGroups, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountGroupSvc).DeleteAccountGroup
		},
	})
}
