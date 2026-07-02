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

// Request to read a single email domain.
type GetEmailDomainRequest struct {
	// Email domain ID.
	ID string `path:"id" validate:"required"`
}

// Returns a single email domain owned by the account.
type GetEmailDomainEndpoint struct{}

func (e *GetEmailDomainEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetEmailDomainRequest, *apiresource.EmailDomain] {
	return (&apiendpoint.APIEndpoint[*GetEmailDomainRequest, *apiresource.EmailDomain]{
		Title:               "Get Email Domain",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-domains/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailDomain,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetEmailDomainRequest) (*apiresource.EmailDomain, *apierror.APIError) {
			return svc.(EmailBridgeSvc).GetDomain
		},
	})
}
