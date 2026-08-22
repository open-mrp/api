package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to activate an account user.
type ActivateAccountUserRequest struct {
	// ID of the account user to activate.
	AccountUserID string `path:"id" validate:"required"`
}

// Activates a disabled or removed account user, restoring their access to the account you are acting in.
//
// Reactivation consumes a seat, so the request fails if the account is at its seat limit. Activating an already-active user is a no-op.
type ActivateAccountUserEndpoint struct{}

func (e *ActivateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*ActivateAccountUserRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ActivateAccountUserRequest, *apiresource.EmptyResource]{
		Title:               "Activate Account User",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users/{id}/actions/activate",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ActivateAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).ActivateAccountUser
		},
	})
}
