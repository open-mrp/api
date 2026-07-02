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

// Request to create a conversation.
type CreateConversationRequest struct {
	// The kind of conversation to create.
	Type constants.ConversationType `json:"type" validate:"required"`
	// The other participant(s).
	//
	// For a direct message, exactly one account_user ID. For a group, the members to add — optional when `group_id` seeds the roster.
	ParticipantAccountUserIDs []string `json:"participant_account_user_ids,omitzero" validate:"omitempty,min=1"`
	// Seed a group conversation from a reusable roster.
	//
	// The roster's current members are copied into this conversation (in addition to any `participant_account_user_ids`); the conversation is independent afterward. Ignored for direct messages.
	GroupID field.Optional[string] `json:"group_id,omitzero"`
	// Title for a group conversation.
	//
	// Ignored for direct messages.
	Title field.Optional[string] `json:"title,omitzero"`
	// The type of business record to anchor this conversation to.
	TopicResourceType field.Optional[constants.ObjectType] `json:"topic_resource_type,omitzero"`
	// The id of the business record to anchor this conversation to.
	TopicResourceID field.Optional[string] `json:"topic_resource_id,omitzero"`
}

var sampleCreateConversationTitle = "Order #1042 — shipping question"

var sampleCreateConversationRequest = &CreateConversationRequest{
	Type:                      constants.ConversationTypeGroup,
	ParticipantAccountUserIDs: []string{apiresource.SampleAccountUserID},
	GroupID:                   field.Some(apiresource.SampleMessagingGroupID),
	Title:                     field.Some(sampleCreateConversationTitle),
	TopicResourceType:         field.Some(constants.ObjectTypeSalesOrder),
	TopicResourceID:           field.Some(apiresource.SampleSalesOrderID),
}

func (*CreateConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateConversationRequest)
}

// Starts a conversation between participants.
type CreateConversationEndpoint struct{}

func (e *CreateConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*CreateConversationRequest, *apiresource.Conversation]{
		Title:               "Create Conversation",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		AgentTool:           true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).CreateConversation
		},
		LocationFunc: func(resp *apiresource.Conversation) string {
			return "/v1/messaging/conversations/" + resp.ID
		},
	})
}
