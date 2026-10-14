package signingpinep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to clear an account user's signing PIN.
type ClearSigningPinRequest struct {
	// ID of the account user, in the caller's account, whose PIN to clear.
	AccountUserID string `path:"id" validate:"required"`
}

// Removes an account user's signing PIN, so they must set a new one before they can sign or check in.
//
// Clearing a person who has no PIN succeeds.
type ClearSigningPinEndpoint struct{}

func (e *ClearSigningPinEndpoint) Materialize() *apiendpoint.APIEndpoint[*ClearSigningPinRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ClearSigningPinRequest, *apiresource.EmptyResource]{
		Title:               "Clear Signing PIN",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users/{id}/actions/clear-signing-pin",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ClearSigningPinRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SigningPinSvc).ClearSigningPin
		},
	})
}
