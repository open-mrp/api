package conversationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a conversation.
type DeleteConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Deletes a conversation that only you and agents are in, permanently.
//
// Conversations on legal hold, or where an agent changed records, can only be archived; so can conversations with other people, which you can also leave. Those return 409 `resource_in_use`.
type DeleteConversationEndpoint struct{}

func (e *DeleteConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteConversationRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteConversationRequest, *object.Deleted]{
		Title:             "Delete Conversation",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/messaging/conversations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeConversation,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteConversationRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ConversationSvc).DeleteConversation, constants.ObjectTypeConversation, "id")
		},
	})
}
