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
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to create a conversation.
type CreateConversationRequest struct {
	// Who else is in the conversation. You are always in it and don't list yourself.
	//
	// With exactly one other person, and no `group_id` or `record`, the conversation is a `direct_message`; otherwise it's a `group`, which you own. Everyone added is notified.
	Participants []ConversationParticipantInput `json:"participants,omitzero" validate:"omitempty,dive"`
	// A messaging group whose current members join the conversation as well. Later changes to the group don't affect the conversation.
	GroupID field.Optional[string] `json:"group_id,omitzero"`
	// The conversation's title. Without one, clients show the participants.
	Title field.Optional[string] `json:"title,omitzero"`
	// The record the conversation is about, such as an order. It appears in the conversation's `related.record`, and the conversation is listed with that record's discussions.
	Record field.Optional[apiresource.RecordInput] `json:"record,omitzero"`
	// A first message to post as soon as the conversation exists.
	Message field.Optional[CreateConversationMessageInput] `json:"message,omitzero"`
	// A conversation this one picks up from.
	//
	// Agents start from a summary of it instead of its full history. You must be in it.
	Continues field.Optional[string] `json:"continues,omitzero"`
}

// A person to add to a new conversation.
type ConversationParticipantInput struct {
	// The account user to add.
	AccountUserID string `json:"account_user_id" validate:"required"`
	// The role they start with in a group; a direct message has no roles. Defaults to `member`.
	//
	// - `owner`: can rename or delete the conversation and manage members and roles, alongside you.
	// - `admin`: can add and remove members and rename the conversation.
	// - `member`: can post, leave, mute, and react.
	// - `viewer`: read-only access.
	Role field.Optional[constants.ParticipantRole] `json:"role,omitzero"`
}

// The first message of a new conversation.
type CreateConversationMessageInput struct {
	// The message text.
	Body string `json:"body" validate:"required,max=10000"`
}

var sampleCreateConversationRequest = &CreateConversationRequest{
	Participants: []ConversationParticipantInput{{AccountUserID: apiresource.SampleAccountUserID, Role: field.Some(constants.ParticipantRoleAdmin)}},
	GroupID:      field.Some(apiresource.SampleMessagingGroupID),
	Title:        field.Some("Order #1042 shipping question"),
	Record:       field.Some(*apiresource.SampleRecordInput),
	Message:      field.Some(CreateConversationMessageInput{Body: "Can we ship this today?"}),
	Continues:    field.Some(apiresource.SampleConversationID),
}

func (*CreateConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateConversationRequest)
}

// Starts a new conversation.
//
// Every call creates a new conversation, even with the same participants. Find an existing one with the `participant_ids[]` filter. A direct message is refused when either person has blocked the other, and creating conversations is rate limited per user.
type CreateConversationEndpoint struct{}

func (e *CreateConversationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateConversationRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*CreateConversationRequest, *apiresource.Conversation]{
		Title:             "Create Conversation",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/messaging/conversations",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeConversation,
		IncludeConfig:     conversationIncludeConfig(),
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateConversationRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).CreateConversation
		},
		LocationFunc: func(resp *apiresource.Conversation) string {
			return "/v1/messaging/conversations/" + resp.ID
		},
	})
}
