package portaldomainep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to re-check a portal domain's DNS configuration.
type VerifyPortalDomainRequest struct {
	// Portal domain ID.
	ID string `path:"id" validate:"required"`
}

// Re-checks the domain's DNS configuration and flips it to `verified` once the published records are confirmed.
//
// Returns the updated domain (still `pending` if DNS has not propagated yet) along with the currently required DNS records.
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
