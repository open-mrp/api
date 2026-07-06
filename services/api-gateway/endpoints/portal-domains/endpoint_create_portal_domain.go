package portaldomainep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to connect a custom domain to the account's customer portal.
type CreatePortalDomainRequest struct {
	// The fully-qualified domain name to connect (e.g. `shop.acme.com`). Subdomains are recommended; apex domains are supported via an A record.
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
// Each account can have one custom domain. The domain starts in `pending` until its DNS is verified.
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
