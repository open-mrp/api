package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to rename a conversation.
type UpdateConversationRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The group conversation's new display title.
	//
	// Send `null` to clear the title and leave the conversation unnamed.
	Title field.Clearable[string] `json:"title,omitzero"`
}

var sampleUpdateConversationRequest = &UpdateConversationRequest{
	ConversationID: apiresource.SampleConversationID,
	Title:          field.Set("Fulfillment war room"),
}

func (*UpdateConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateConversationRequest)
}

// Renames a group conversation.
//
// Only an owner or admin of the conversation can rename it, and direct messages cannot be renamed.
type UpdateConversationEndpoint struct{}

func (e *UpdateConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*UpdateConversationRequest, *apiresource.Conversation]{
		Title:               "Update Conversation",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).UpdateConversation
		},
	})
}
