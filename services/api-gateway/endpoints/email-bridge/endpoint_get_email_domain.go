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
)

// Request to read a single email domain.
type GetEmailDomainRequest struct {
	// Email domain ID.
	ID string `path:"id" validate:"required"`
}

// Returns a single email domain owned by the account.
type GetEmailDomainEndpoint struct{}

func (e *GetEmailDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetEmailDomainRequest, *apiresource.EmailDomain] {
	return (&apiendpoint.APIEndpoint[*GetEmailDomainRequest, *apiresource.EmailDomain]{
		Title:             "Get Email Domain",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/messaging/email-domains/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeEmailDomain,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError) {
			return svc.(EmailBridgeSvc).GetDomain
		},
	})
}
