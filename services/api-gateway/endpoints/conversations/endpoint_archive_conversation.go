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

// Request for the Archive Conversation action.
type ArchiveConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
}

// Archives a conversation for the whole account rather than just for the caller.
//
// Only an owner or admin of the conversation can archive it, and direct messages cannot be archived. An archived customer-facing case leaves the working support inbox and is returned only by the archived view.
type ArchiveConversationEndpoint struct{}

func (e *ArchiveConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*ArchiveConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*ArchiveConversationRequest, *apiresource.Conversation]{
		Title:               "Archive Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/archive",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ArchiveConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).ArchiveConversation
		},
	})
}
