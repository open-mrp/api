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

// Request for the Unmute Conversation action.
type UnmuteConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Restores notifications for a conversation the caller had muted.
type UnmuteConversationEndpoint struct{}

func (e *UnmuteConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnmuteConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*UnmuteConversationRequest, *apiresource.Conversation]{
		Title:               "Unmute Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/unmute",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnmuteConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).UnmuteConversation
		},
	})
}
