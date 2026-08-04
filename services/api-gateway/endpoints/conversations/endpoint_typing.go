package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request for the Send Typing Indicator action.
type TypingRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Broadcasts an ephemeral "typing" indicator to a conversation's live subscribers.
//
// The signal is not persisted; clients reconcile from message history, never from typing events.
type TypingEndpoint struct{}

func (e *TypingEndpoint) Materialize() *apiendpoint.APIEndpoint[*TypingRequest, *apiresource.MessageResource] {
	return (&apiendpoint.APIEndpoint[*TypingRequest, *apiresource.MessageResource]{
		Title:               "Send Typing Indicator",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/typing",
		SuccessStatusCode:   http.StatusAccepted,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessage,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *TypingRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(ConversationSvc).SendTyping
		},
	})
}
