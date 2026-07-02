package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the account's registered email domains.
type ListEmailDomainsRequest struct{}

// Returns the account's registered email domains.
type ListEmailDomainsEndpoint struct{}

func (e *ListEmailDomainsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEmailDomainsRequest, *apiresource.List[apiresource.EmailDomain]] {
	return (&apiendpoint.APIEndpoint[*ListEmailDomainsRequest, *apiresource.List[apiresource.EmailDomain]]{
		Title:               "List Email Domains",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-domains",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEmailDomainsRequest) (*apiresource.List[apiresource.EmailDomain], *apierror.APIError) {
			return svc.(EmailBridgeSvc).ListDomains
		},
	})
}
