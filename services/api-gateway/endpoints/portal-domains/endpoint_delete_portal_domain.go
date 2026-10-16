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

// Deletes a portal domain.
//
// The portal stops answering on the domain at once and is served from its default address again. You can then connect a different domain and remove the DNS records.
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
		ObjectType:        constants.ObjectTypePortalDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePortalDomainRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PortalDomainSvc).DeletePortalDomain, constants.ObjectTypePortalDomain, "id")
		},
	})
}
