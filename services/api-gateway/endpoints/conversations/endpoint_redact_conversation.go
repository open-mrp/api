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

// Request to redact a conversation.
//
// Strips the body and attachments from every message while keeping the message rows as an audit shell. Refused while the conversation is under legal hold.
type RedactConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Permanently redacts the content of every message in a conversation (GDPR right-to-erasure).
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
