package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to disable an account user.
type DisableAccountUserRequest struct {
	// ID of the account user to disable.
	AccountUserID string `path:"id" validate:"required"`
}

// Disables (locks) an account user.
//
// Disabled users cannot access the account and their active sessions are revoked, but the membership and its role assignment are kept so access can be restored with the activate action. Disabling frees the seat the user occupied. Admin users cannot be disabled, you cannot disable yourself, and removed users must be activated before they can be disabled. Disabling an already-disabled user is a no-op.
type DisableAccountUserEndpoint struct{}

func (e *DisableAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*DisableAccountUserRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DisableAccountUserRequest, *apiresource.EmptyResource]{
		Title:               "Disable Account User",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users/{id}/actions/disable",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DisableAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).DisableAccountUser
		},
	})
}
