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

// Request to delete an email inbox.
type DeleteEmailInboxRequest struct {
	// Email inbox ID.
	ID string `path:"id" validate:"required"`
}

// Removes an email inbox.
//
// Mail sent to its address is no longer routed. Conversations the inbox already opened are kept, but replies can no longer be sent on them, so disable the inbox instead of deleting it if you still need to answer open threads.
type DeleteEmailInboxEndpoint struct{}

func (e *DeleteEmailInboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteEmailInboxRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteEmailInboxRequest, *apiresource.EmptyResource]{
		Title:               "Delete Email Inbox",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-inboxes/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailInbox,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteEmailInboxRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(EmailBridgeSvc).DeleteInbox
		},
	})
}
