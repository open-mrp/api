package emailbridgeep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to re-check a domain's DKIM verification status.
type VerifyEmailDomainRequest struct {
	// Email domain ID.
	ID string `path:"id" validate:"required"`
}

// Verifies an email domain now, instead of waiting for the automatic check.
//
// Pending domains are checked automatically, so you don't need to call this. It is safe to call repeatedly: a domain whose `dns_records` aren't visible yet stays `pending`, and an already-verified domain is returned as is.
type VerifyEmailDomainEndpoint struct{}

func (e *VerifyEmailDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*VerifyEmailDomainRequest, *apiresource.EmailDomain] {
	return (&apiendpoint.APIEndpoint[*VerifyEmailDomainRequest, *apiresource.EmailDomain]{
		Title:             "Verify Email Domain",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/messaging/email-domains/{id}/actions/verify",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeEmailDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *VerifyEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError) {
			return svc.(EmailBridgeSvc).VerifyDomain
		},
	})
}
