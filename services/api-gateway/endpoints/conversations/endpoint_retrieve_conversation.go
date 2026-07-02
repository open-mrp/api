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

// Request to retrieve a single conversation the caller participates in.
type RetrieveConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Returns one conversation (with participants) the caller belongs to.
type RetrieveConversationEndpoint struct{}

func (e *RetrieveConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*RetrieveConversationRequest, *apiresource.Conversation]{
		Title:               "Retrieve Conversation",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).GetConversation
		},
	})
}
