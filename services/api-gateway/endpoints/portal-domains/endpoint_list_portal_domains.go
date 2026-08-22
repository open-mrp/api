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

// Request to list the account's portal domains.
type ListPortalDomainsRequest struct{}

// Lists the account's portal domains.
//
// An account can only hold one custom portal domain, so this returns either zero or one entry. Reading it is the usual way to discover whether a domain is connected and what state it is in.
type ListPortalDomainsEndpoint struct{}

func (e *ListPortalDomainsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPortalDomainsRequest, *apiresource.List[apiresource.PortalDomain]] {
	return (&apiendpoint.APIEndpoint[*ListPortalDomainsRequest, *apiresource.List[apiresource.PortalDomain]]{
		Title:               "List Portal Domains",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/settings/portal-domains",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypePortalDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPortalDomainsRequest) (*apiresource.List[apiresource.PortalDomain], *apierror.APIError) {
			return svc.(PortalDomainSvc).ListPortalDomains
		},
	})
}
