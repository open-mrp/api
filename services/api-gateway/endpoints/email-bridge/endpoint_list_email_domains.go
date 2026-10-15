package emailbridgeep

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

// Request to list the account's registered email domains.
type ListEmailDomainsRequest struct {
	object.ListRequest
}

// Returns a list of your email domains.
//
// The domains are sorted by creation date, newest first.
type ListEmailDomainsEndpoint struct{}

func (e *ListEmailDomainsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEmailDomainsRequest, *apiresource.List[apiresource.EmailDomain]] {
	return (&apiendpoint.APIEndpoint[*ListEmailDomainsRequest, *apiresource.List[apiresource.EmailDomain]]{
		Title:             "List Email Domains",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/messaging/email-domains",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeEmailDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEmailDomainsRequest) (*apiresource.List[apiresource.EmailDomain], *apierror.APIError) {
			return svc.(EmailBridgeSvc).ListDomains
		},
	})
}
