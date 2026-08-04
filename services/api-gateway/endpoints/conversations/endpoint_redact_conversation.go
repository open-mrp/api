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

// Request for the Redact Conversation action.
type RedactConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Permanently erases the content of every message in a conversation, for right-to-erasure requests.
//
// Message bodies are cleared and attachments are deleted from storage, leaving the messages behind as an empty audit shell. This cannot be undone, and it is refused while the conversation is under legal hold.
type RedactConversationEndpoint struct{}

func (e *RedactConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*RedactConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*RedactConversationRequest, *apiresource.Conversation]{
		Title:               "Redact Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/redact",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RedactConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).RedactConversation
		},
	})
}
