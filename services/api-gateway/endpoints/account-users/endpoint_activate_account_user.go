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
// Reactivating a user in your own account consumes a seat, so the request fails if your plan is at its seat limit; users of a customer or supplier account you manage take no seat. Activating an already-active user is a no-op.
type ActivateAccountUserEndpoint struct{}

func (e *ActivateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*ActivateAccountUserRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ActivateAccountUserRequest, *apiresource.EmptyResource]{
		Title:                   "Activate Account User",
		Method:                  http.MethodPut,
		ContentType:             "application/json",
		Route:                   "/v1/identity/account-users/{id}/actions/activate",
		SuccessStatusCode:       http.StatusOK,
		Public:                  true,
		AgentTool:               true,
		RequiredPermissions:     []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}},
		CounterpartyPermissions: apiendpoint.Counterparties(types.ActionUpdate),
		Preview:                 true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ActivateAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AccountUserSvc).ActivateAccountUser
		},
	})
}
