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
		Title:             "Retrieve Portal Domain",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/portal-domains/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypePortalDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
			return svc.(PortalDomainSvc).GetPortalDomain
		},
	})
}
