package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a conversation.
type CreateConversationRequest struct {
	// The kind of conversation to create.
	//
	// - `direct_message`: a 1:1 thread with exactly one other user. Addressing yourself is allowed and gives you a private notes thread.
	// - `group`: a named thread with any number of user and agent members.
	//
	// `system` channels are created by the platform and cannot be requested here.
	Type constants.ConversationType `json:"type" validate:"required"`
	// The other participants to add.
	//
	// For a direct message, exactly one account user. For a group, the members to seed — these can be omitted when `group_id` supplies a roster, or when the conversation is anchored to a topic resource, since a record discussion may start solo and pull people in later.
	//
	// The caller is always a participant and does not need to be listed; on a group they become its owner and every other member seeded at creation is notified.
	ParticipantAccountUserIDs []string `json:"participant_account_user_ids,omitzero" validate:"omitempty,dive,required"`
	// Members to seat in a group with the role each starts with.
	//
	// Each account user listed here joins the group as those in `participant_account_user_ids` do, then takes the role given, exactly as if you set it on the new participant: the change is announced in the thread and recorded on the participant's history. A user also listed in `participant_account_user_ids` or on the roster takes the role given here. You cannot list yourself (you own the group) or the same user twice, and a direct message takes no roles.
	Participants []ConversationParticipantInput `json:"participants,omitzero" validate:"omitempty,dive"`
	// Seed a group conversation from a reusable roster.
	//
	// The roster's current members are copied into this conversation (in addition to any `participant_account_user_ids`); the conversation is independent afterward. Ignored for direct messages.
	GroupID field.Optional[string] `json:"group_id,omitzero"`
	// Title for a group conversation.
	//
	// A direct message is identified by its participants rather than by a title.
	Title field.Optional[string] `json:"title,omitzero"`
	// The type of business record to anchor this conversation to.
	//
	// An anchored conversation is returned when conversations are listed for that record, which is how a discussion shows up on an order or invoice.
	TopicResourceType field.Optional[constants.ObjectType] `json:"topic_resource_type,omitzero"`
	// The id of the business record to anchor this conversation to.
	TopicResourceID field.Optional[string] `json:"topic_resource_id,omitzero"`
}

// A member to seat in a new group conversation, with the role they start with.
type ConversationParticipantInput struct {
	// The account user to add.
	AccountUserID string `json:"account_user_id" validate:"required"`
	// The role the member starts with.
	//
	// - `owner`: can rename or delete the conversation and manage members and roles, alongside you.
	// - `admin`: can add and remove members and rename the conversation.
	// - `member`: can post, leave, mute, and react.
	// - `viewer`: read-only access.
	Role constants.ParticipantRole `json:"role" validate:"required"`
}

var sampleCreateConversationTitle = "Order #1042 — shipping question"

var sampleCreateConversationRequest = &CreateConversationRequest{
	Type:                      constants.ConversationTypeGroup,
	ParticipantAccountUserIDs: []string{apiresource.SampleAccountUserID},
	Participants:              []ConversationParticipantInput{{AccountUserID: apiresource.SampleAccountUserID, Role: constants.ParticipantRoleAdmin}},
	GroupID:                   field.Some(apiresource.SampleMessagingGroupID),
	Title:                     field.Some(sampleCreateConversationTitle),
	TopicResourceType:         field.Some(constants.ObjectTypeSalesOrder),
	TopicResourceID:           field.Some(apiresource.SampleSalesOrderID),
}

func (*CreateConversationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateConversationRequest)
}

// Starts a direct message or group conversation.
//
// Requesting a direct message that already exists returns the existing thread instead of creating a duplicate, and a direct message is refused when either user has blocked the other. Conversation creation is rate limited per user.
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
