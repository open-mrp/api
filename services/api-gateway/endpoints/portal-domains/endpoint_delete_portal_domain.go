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

// Request to disconnect a portal domain.
type DeletePortalDomainRequest struct {
	// Portal domain ID.
	ID string `path:"id" validate:"required"`
}

// Disconnects the custom domain from the account's customer portal.
//
// The domain is detached from the serving infrastructure and immediately stops serving the portal.
type DeletePortalDomainEndpoint struct{}

func (e *DeletePortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePortalDomainRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeletePortalDomainRequest, *apiresource.EmptyResource]{
		Title:               "Delete Portal Domain",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/settings/portal-domains/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypePortalDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePortalDomainRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PortalDomainSvc).DeletePortalDomain
		},
	})
}
