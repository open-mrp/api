package messageep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list a conversation's messages.
type ListMessagesRequest struct {
	apiresource.PaginationRequest
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// Filter by lifecycle state.
	//
	// Defaults to `sent` (the conversation timeline); pass `draft` to list the case's open customer-reply drafts, or `scheduled` to list your not-yet-sent scheduled messages in this conversation.
	Status *constants.MessageStatus `query:"status"`
	// Catch-up bound.
	//
	// Only return messages with a sequence greater than this (reconnect sync).
	AfterSequence *int64 `query:"after_sequence"`
}

// Returns a conversation's messages, newest first, keyset-paginated by sequence.
type ListMessagesEndpoint struct{}

func (e *ListMessagesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMessagesRequest, *apiresource.List[apiresource.Message]] {
	return (&apiendpoint.APIEndpoint[*ListMessagesRequest, *apiresource.List[apiresource.Message]]{
		Title:               "List Messages",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/messages",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeChatMessage,
		IncludeConfig:       messageIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMessagesRequest) (*apiresource.List[apiresource.Message], *apierror.APIError) {
			return svc.(MessageSvc).ListMessages
		},
	})
}
