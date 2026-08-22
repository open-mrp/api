package portaldomainep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to re-check a portal domain's DNS configuration.
type VerifyPortalDomainRequest struct {
	// Portal domain ID.
	ID string `path:"id" validate:"required"`
}

// Re-checks a portal domain against the serving provider and advances its status.
//
// Run this after publishing the DNS records, and keep polling it: the domain stays `pending` while its records are missing or misconfigured, moves to `securing` once they are correct and its TLS certificate is being issued, and reaches `verified` only once that certificate is live and the portal answers on the domain. The response carries the updated domain along with the records still required. Verifying an already-verified domain returns it unchanged.
type VerifyPortalDomainEndpoint struct{}

func (e *VerifyPortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*VerifyPortalDomainRequest, *apiresource.PortalDomain] {
	return (&apiendpoint.APIEndpoint[*VerifyPortalDomainRequest, *apiresource.PortalDomain]{
		Title:               "Verify Portal Domain",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/settings/portal-domains/{id}/actions/verify",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypePortalDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *VerifyPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
			return svc.(PortalDomainSvc).VerifyPortalDomain
		},
	})
}
