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

// Request for the Leave Conversation action.
type LeaveConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Removes the caller from a conversation, marking their membership as left.
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
