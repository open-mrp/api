package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to advance the caller's read cursor in a conversation.
type MarkConversationReadRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// Mark every message up to and including this sequence number as read.
	//
	// A sequence past the conversation's latest message is clamped to it, and the read position never moves backwards, so replaying an older value is harmless.
	UpToSequence int64 `json:"up_to_sequence" validate:"required"`
}

var sampleMarkConversationReadRequest = &MarkConversationReadRequest{
	ConversationID: apiresource.SampleConversationID,
	UpToSequence:   42,
}

func (*MarkConversationReadRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleMarkConversationReadRequest)
}

// Advances the caller's read position in a conversation and returns it with the recalculated unread count.
//
// Reading also dismisses the caller's outstanding notifications for this conversation, and updates the read receipt the other participants see.
type MarkConversationReadEndpoint struct{}

func (e *MarkConversationReadEndpoint) Materialize() *apiendpoint.APIEndpoint[*MarkConversationReadRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*MarkConversationReadRequest, *apiresource.Conversation]{
		Title:               "Mark Conversation Read",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/read",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		AgentTool:           true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkConversationReadRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).MarkConversationRead
		},
	})
}
