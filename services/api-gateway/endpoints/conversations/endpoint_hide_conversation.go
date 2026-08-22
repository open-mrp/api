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

// Request for the Hide Conversation action.
type HideConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Hides a conversation from the caller's own list without affecting other participants.
//
// The caller stays a member and keeps receiving notifications; the conversation simply stops appearing in their list until they unhide it, and new messages do not bring it back on their own. The owner of a conversation cannot hide it.
type HideConversationEndpoint struct{}

func (e *HideConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*HideConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*HideConversationRequest, *apiresource.Conversation]{
		Title:               "Hide Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/hide",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		AgentTool:           false,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *HideConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).HideConversation
		},
	})
}
