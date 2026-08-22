package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request for the Leave Conversation action.
type LeaveConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Removes the caller from a conversation.
//
// An owner cannot leave — hand ownership to someone else first. Leaving posts a "left the conversation" note to the thread and hides the conversation for the caller, who can still read it back but can no longer post.
type LeaveConversationEndpoint struct{}

func (e *LeaveConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*LeaveConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*LeaveConversationRequest, *apiresource.Conversation]{
		Title:               "Leave Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/leave",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		AgentTool:           false,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *LeaveConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).LeaveConversation
		},
	})
}
