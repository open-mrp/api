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

// Request to delete an email inbox.
type DeleteEmailInboxRequest struct {
	// Email inbox ID.
	ID string `path:"id" validate:"required"`
}

// Removes an email inbox.
//
// Mail sent to its address is no longer routed. Conversations the inbox already opened are kept, but replies can no longer be sent on them, so disable the inbox instead of deleting it if you still need to answer open threads.
type DeleteEmailInboxEndpoint struct{}

func (e *DeleteEmailInboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteEmailInboxRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteEmailInboxRequest, *object.Deleted]{
		Title:             "Delete Email Inbox",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/messaging/email-inboxes/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeEmailInbox,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteEmailInboxRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(EmailBridgeSvc).DeleteInbox, constants.ObjectTypeEmailInbox, "id")
		},
	})
}
