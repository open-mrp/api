package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to re-check a domain's DKIM verification status.
type VerifyEmailDomainRequest struct {
	// Email domain ID.
	ID string `path:"id" validate:"required"`
}

// Checks whether the domain's DKIM records have been published and marks it `verified` once they are confirmed.
//
// Call this after publishing the DKIM records returned at registration. It is safe to call repeatedly: a domain whose records are not visible yet is returned unchanged in `pending`, and an already-verified domain is returned as-is without re-checking. DNS propagation can take a while, so expect to poll.
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
