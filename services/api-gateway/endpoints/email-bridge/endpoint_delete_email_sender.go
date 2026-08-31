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

// Request to clear the account's configured email sender.
type DeleteEmailSenderRequest struct{}

// Clears the configured sending address, returning your customer-facing email to the platform address.
type DeleteEmailSenderEndpoint struct{}

func (e *DeleteEmailSenderEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteEmailSenderRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteEmailSenderRequest, *apiresource.EmptyResource]{
		Title:               "Delete Email Sender",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-sender",
		SuccessStatusCode:   http.StatusNoContent,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailSender,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteEmailSenderRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(EmailBridgeSvc).DeleteSender
		},
	})
}
