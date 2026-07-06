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

// Returns a single portal domain, including the DNS records the customer must publish.
type GetPortalDomainEndpoint struct{}

func (e *GetPortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPortalDomainRequest, *apiresource.PortalDomain] {
	return (&apiendpoint.APIEndpoint[*GetPortalDomainRequest, *apiresource.PortalDomain]{
		Title:               "Get Portal Domain",
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
