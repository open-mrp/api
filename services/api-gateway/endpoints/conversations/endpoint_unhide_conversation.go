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

// Request for the Unhide Conversation action.
type UnhideConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Restores a conversation the caller had hidden back to their own list.
type UnhideConversationEndpoint struct{}

func (e *UnhideConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnhideConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*UnhideConversationRequest, *apiresource.Conversation]{
		Title:               "Unhide Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/unhide",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnhideConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).UnhideConversation
		},
	})
}
