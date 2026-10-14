package conversationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
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
		Title:             "Unlink Record",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/messaging/conversations/{id}/links/{link_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveConversationLinkRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ConversationSvc).RemoveConversationLink
		},
	})
}
