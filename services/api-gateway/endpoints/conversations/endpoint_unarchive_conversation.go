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

// Request for the Unarchive Conversation action.
type UnarchiveConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Returns an archived conversation to the active state for the whole account.
//
// Only an owner or admin of the conversation can unarchive it. An unarchived customer-facing case comes back to the working support inbox, and participants who had separately hidden the conversation still see it hidden until they unhide it themselves.
type UnarchiveConversationEndpoint struct{}

func (e *UnarchiveConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnarchiveConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*UnarchiveConversationRequest, *apiresource.Conversation]{
		Title:               "Unarchive Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/unarchive",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnarchiveConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).UnarchiveConversation
		},
	})
}
