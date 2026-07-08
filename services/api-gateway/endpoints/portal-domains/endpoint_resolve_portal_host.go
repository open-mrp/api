package portaldomainep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to resolve a custom portal host to its account.
type ResolvePortalHostRequest struct {
	// The request host (custom domain) to resolve, e.g. `shop.acme.com`.
	Domain string `path:"domain" validate:"required"`
}

// Resolves a verified custom portal domain to the public profile of the account it serves.
//
// This endpoint does not require authentication; the frontend uses it to map a request host to a customer portal. Unverified or unknown domains return a 404.
type ResolvePortalHostEndpoint struct{}

func (e *ResolvePortalHostEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResolvePortalHostRequest, *apiresource.PublicAccount] {
	return (&apiendpoint.APIEndpoint[*ResolvePortalHostRequest, *apiresource.PublicAccount]{
		Title:             "Resolve Portal Host",
		Method:            http.MethodGet,
		Route:             "/v1/settings/portal-hosts/{domain}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ObjectType:        constants.ObjectTypePublicAccount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ResolvePortalHostRequest) (*apiresource.PublicAccount, *apierror.APIError) {
			return svc.(PortalDomainSvc).ResolvePortalHost
		},
	})
}
