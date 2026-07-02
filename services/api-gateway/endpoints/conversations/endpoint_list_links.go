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

// Request to list a conversation's business-record links.
type ListConversationLinksRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Returns the business records linked to a conversation.
type ListConversationLinksEndpoint struct{}

func (e *ListConversationLinksEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListConversationLinksRequest, *apiresource.List[apiresource.ConversationLink]] {
	return (&apiendpoint.APIEndpoint[*ListConversationLinksRequest, *apiresource.List[apiresource.ConversationLink]]{
		Title:               "List Links",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/links",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversationLink,
		IncludeConfig:       conversationLinkIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListConversationLinksRequest) (*apiresource.List[apiresource.ConversationLink], *apierror.APIError) {
			return svc.(ConversationSvc).ListConversationLinks
		},
	})
}
