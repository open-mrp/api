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
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to connect a custom domain to the account's customer portal.
type CreatePortalDomainRequest struct {
	// The fully qualified domain name, such as `shop.acme.com`.
	//
	// A subdomain is routed with a `CNAME` record and an apex domain with an `A` record. The name is lowercased and a trailing dot is dropped. OpenMRP hostnames are rejected.
	Domain string `json:"domain" validate:"required"`
}

var sampleCreatePortalDomainRequest = &CreatePortalDomainRequest{
	Domain: "shop.acme.com",
}

func (*CreatePortalDomainRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePortalDomainRequest)
}

// Adds a custom domain to your customer portal.
//
// An account has one portal domain: adding a second, or a domain another account uses, is a 409. The domain starts `pending`; publish its `dns_records` at your DNS provider and OpenMRP checks it automatically.
type CreatePortalDomainEndpoint struct{}

func (e *CreatePortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePortalDomainRequest, *apiresource.PortalDomain] {
	return (&apiendpoint.APIEndpoint[*CreatePortalDomainRequest, *apiresource.PortalDomain]{
		Title:             "Create Portal Domain",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/settings/portal-domains",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePortalDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
			return svc.(PortalDomainSvc).CreatePortalDomain
		},
	})
}
