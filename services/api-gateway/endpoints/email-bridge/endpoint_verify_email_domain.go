package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to re-check a domain's DKIM verification status.
type VerifyEmailDomainRequest struct {
	// Email domain ID.
	ID string `path:"id" validate:"required"`
}

// Re-polls the provider and flips the domain to `verified` once its DKIM records are confirmed.
//
// Returns the updated domain (still `pending` if not yet confirmed).
type VerifyEmailDomainEndpoint struct{}

func (e *VerifyEmailDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*VerifyEmailDomainRequest, *apiresource.EmailDomain] {
	return (&apiendpoint.APIEndpoint[*VerifyEmailDomainRequest, *apiresource.EmailDomain]{
		Title:               "Verify Email Domain",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-domains/{id}/actions/verify",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *VerifyEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError) {
			return svc.(EmailBridgeSvc).VerifyDomain
		},
	})
}
