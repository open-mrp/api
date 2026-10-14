package emailbridgeep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to clear the account's configured email sender.
type DeleteEmailSenderRequest struct{}

// Deletes your email sender.
//
// Your order, invoice and statement emails then send from the platform address. Returns 404 when no sender is set up.
type DeleteEmailSenderEndpoint struct{}

func (e *DeleteEmailSenderEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteEmailSenderRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteEmailSenderRequest, *object.Deleted]{
		Title:             "Delete Email Sender",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/messaging/email-sender",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeEmailSender,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteEmailSenderRequest) (*object.Deleted, *apierror.APIError) {
			return svc.(EmailBridgeSvc).DeleteSender
		},
	})
}
