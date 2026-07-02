package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to remove a business-record link from a conversation.
type RemoveConversationLinkRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The id of the link to remove.
	LinkID string `path:"link_id" validate:"required"`
}

// Removes a business-record link from a conversation.
type RemoveConversationLinkEndpoint struct{}

func (e *RemoveConversationLinkEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveConversationLinkRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RemoveConversationLinkRequest, *apiresource.EmptyResource]{
		Title:               "Unlink Record",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/links/{link_id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveConversationLinkRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ConversationSvc).RemoveConversationLink
		},
	})
}
