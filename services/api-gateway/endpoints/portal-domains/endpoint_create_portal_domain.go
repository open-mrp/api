package portaldomainep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to connect a custom domain to the account's customer portal.
type CreatePortalDomainRequest struct {
	// The fully-qualified domain name to connect (e.g. `shop.acme.com`).
	//
	// A subdomain such as `shop.acme.com` is routed with a CNAME record and an apex domain such as `acme.com` with an A record; either way the records to publish come back on the response. The value is lowercased and any trailing dot is stripped before it is stored, and OpenMRP-owned hostnames are rejected.
	Domain string `json:"domain" validate:"required"`
}

var sampleCreatePortalDomainRequest = &CreatePortalDomainRequest{
	Domain: "shop.acme.com",
}

func (*CreatePortalDomainRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePortalDomainRequest)
}

// Connects a custom domain to the account's customer portal and returns the DNS records to publish.
//
// An account can only have one custom domain at a time: adding a second one — or claiming a domain another account already uses — returns a conflict error. The new domain starts in `pending`; publish the returned records at your DNS provider, then run the verify action to move it towards serving.
type CreatePortalDomainEndpoint struct{}

func (e *CreatePortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePortalDomainRequest, *apiresource.PortalDomain] {
	return (&apiendpoint.APIEndpoint[*CreatePortalDomainRequest, *apiresource.PortalDomain]{
		Title:               "Create Portal Domain",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/settings/portal-domains",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypePortalDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
			return svc.(PortalDomainSvc).CreatePortalDomain
		},
	})
}
