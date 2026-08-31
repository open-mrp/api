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

// Request to read the account's configured email sender.
type GetEmailSenderRequest struct{}

// Returns the address your order, invoice, and statement emails are sent from, or 404 when none is configured and that mail sends from the platform address.
type GetEmailSenderEndpoint struct{}

func (e *GetEmailSenderEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetEmailSenderRequest, *apiresource.EmailSender] {
	return (&apiendpoint.APIEndpoint[*GetEmailSenderRequest, *apiresource.EmailSender]{
		Title:               "Get Email Sender",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-sender",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailSender,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetEmailSenderRequest) (*apiresource.EmailSender, *apierror.APIError) {
			return svc.(EmailBridgeSvc).GetSender
		},
	})
}
