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

// Request to retrieve a portal domain by ID.
type GetPortalDomainRequest struct {
	// Portal domain ID.
	ID string `path:"id" validate:"required"`
}

// Returns a single portal domain, including its current status and the DNS records that must be published for it.
//
// Reading a domain never re-checks it with the serving provider — the status is the one recorded when the domain was connected or last verified — so run the verify action to move a `pending` or `securing` domain forward.
type GetPortalDomainEndpoint struct{}

func (e *GetPortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPortalDomainRequest, *apiresource.PortalDomain] {
	return (&apiendpoint.APIEndpoint[*GetPortalDomainRequest, *apiresource.PortalDomain]{
		Title:               "Retrieve Portal Domain",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/settings/portal-domains/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypePortalDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
			return svc.(PortalDomainSvc).GetPortalDomain
		},
	})
}
