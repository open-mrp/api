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
	"github.com/open-mrp/apikit/object"
)

// Request to list portal domains.
type ListPortalDomainsRequest struct {
	object.ListRequest
}

// Returns a list of your account's portal domains.
//
// An account has at most one portal domain, so the list holds zero or one.
type ListPortalDomainsEndpoint struct{}

func (e *ListPortalDomainsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPortalDomainsRequest, *apiresource.List[apiresource.PortalDomain]] {
	return (&apiendpoint.APIEndpoint[*ListPortalDomainsRequest, *apiresource.List[apiresource.PortalDomain]]{
		Title:             "List Portal Domains",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/portal-domains",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePortalDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPortalDomainsRequest) (*apiresource.List[apiresource.PortalDomain], *apierror.APIError) {
			return svc.(PortalDomainSvc).ListPortalDomains
		},
	})
}
