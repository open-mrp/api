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

// Request to disconnect a portal domain.
type DeletePortalDomainRequest struct {
	// Portal domain ID.
	ID string `path:"id" validate:"required"`
}

// Disconnects the custom domain from the account's customer portal.
//
// The domain is detached from the serving infrastructure and immediately stops serving the portal; buyers must go back to the account's default slug-based portal address. Because an account may only hold one custom domain, this is how you free it up to connect a different one. The DNS records you published can then be removed.
type DeletePortalDomainEndpoint struct{}

func (e *DeletePortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePortalDomainRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeletePortalDomainRequest, *object.Deleted]{
		Title:             "Delete Portal Domain",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/settings/portal-domains/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypePortalDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePortalDomainRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PortalDomainSvc).DeletePortalDomain, constants.ObjectTypePortalDomain, "id")
		},
	})
}
