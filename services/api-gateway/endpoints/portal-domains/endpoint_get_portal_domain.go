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
)

// Request to retrieve a portal domain by ID.
type GetPortalDomainRequest struct {
	// Portal domain ID.
	ID string `path:"id" validate:"required"`
}

// Retrieves a portal domain.
//
// The status is as of the last check. Pending domains are checked automatically; Verify checks now.
type GetPortalDomainEndpoint struct{}

func (e *GetPortalDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPortalDomainRequest, *apiresource.PortalDomain] {
	return (&apiendpoint.APIEndpoint[*GetPortalDomainRequest, *apiresource.PortalDomain]{
		Title:             "Retrieve Portal Domain",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/portal-domains/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePortalDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPortalDomainRequest) (*apiresource.PortalDomain, *apierror.APIError) {
			return svc.(PortalDomainSvc).GetPortalDomain
		},
	})
}
