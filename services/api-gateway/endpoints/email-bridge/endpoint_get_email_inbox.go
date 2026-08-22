package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to read a single email inbox.
type GetEmailInboxRequest struct {
	// Email inbox ID.
	ID string `path:"id" validate:"required"`
}

// Returns a single email inbox owned by the account.
type GetEmailInboxEndpoint struct{}

func (e *GetEmailInboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetEmailInboxRequest, *apiresource.EmailInbox] {
	return (&apiendpoint.APIEndpoint[*GetEmailInboxRequest, *apiresource.EmailInbox]{
		Title:               "Get Email Inbox",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-inboxes/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailInbox,
		IncludeConfig:       emailInboxIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError) {
			return svc.(EmailBridgeSvc).GetInbox
		},
	})
}
