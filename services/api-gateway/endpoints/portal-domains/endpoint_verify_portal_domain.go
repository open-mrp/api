package portaldomainep

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

// Request to re-check a portal domain's DNS configuration.
type VerifyPortalDomainRequest struct {
	// Portal domain ID.
	ID string `path:"id" validate:"required"`
}

// Verifies a portal domain now.
//
// Pending domains are checked automatically; run this after publishing the DNS records to skip the wait. The domain moves from `pending` to `securing` once its records are correct, and to `verified` once its certificate is live. Verifying a `verified` domain returns it unchanged.
type VerifyPortalDomainEndpoint struct{}

func (e *VerifyPortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*VerifyPortalDomainRequest, *apiresource.PortalDomain] {
	return (&apiendpoint.APIEndpoint[*VerifyPortalDomainRequest, *apiresource.PortalDomain]{
		Title:             "Verify Portal Domain",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/settings/portal-domains/{id}/actions/verify",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePortalDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *VerifyPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
			return svc.(PortalDomainSvc).VerifyPortalDomain
		},
	})
}
