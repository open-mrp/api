package accountgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete an account group.
type DeleteAccountGroupRequest struct {
	// Account group ID.
	AccountGroupID string `path:"id" validate:"required"`
}

// Deletes an account group.
//
// Deletion fails with a validation error while the group is still in use: a `type_group` that is set as a customer's type cannot be deleted, and no group can be deleted while it grants product line access, backs a volume discount, or is attached to a customer registration flow.
//
// Deleting a `pricing_group` first unassigns it from every customer it was applied to, so those customers immediately stop receiving its pricing.
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
