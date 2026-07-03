package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const (
	SampleConversationID            = "cv_01h9z8q1w2e3r4t5y6u7i8cv"
	SampleConversationParticipantID = "cvpt_01h9z8q1w2e3r4t5y6u7cvpt"
)

// A conversation thread the caller participates in.
type Conversation struct {
	// Conversation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=conversation"`
	// What kind of conversation this is.
	//
	// - `direct_message`: a 1:1 thread between two users.
	// - `group`: a named thread with multiple user or agent members (including customer-facing support cases).
	// - `system`: a system channel that delivers automated account alerts.
	Type constants.ConversationType `json:"type" validate:"required"`
	// Whether this is a team-only conversation (`internal`) or a customer-facing case (`customer`).
	Audience constants.ConversationAudience `json:"audience" validate:"required"`
	// The display title of a group conversation.
	//
	// `null` for direct messages, where the client derives a title from the participants.
	Title *string `json:"title"`
	// The triage lane of a customer-facing case.
	//
	// Only set for customer-audience conversations.
	//
	// - `new`: opened but not yet triaged.
	// - `open`: actively being worked.
	// - `waiting_internal`: blocked on the internal team.
	// - `waiting_external`: blocked on an external reply.
	// - `needs_approval`: a drafted reply is awaiting human approval.
	// - `resolved`: closed out.
	WorkflowStatus *constants.ConversationWorkflowStatus `json:"workflow_status"`
	// The reusable roster this conversation was started from, when one was used.
	//
	// `null` for ad-hoc conversations. Provenance only: members were copied in at creation and are not driven by the roster thereafter.
	Group *MessagingGroup `json:"group" expandable:"true"`
	// The caller's effective status.
	//
	// - `hidden` when the caller has hidden the conversation
	// - otherwise the account-level lifecycle state
	Status constants.ConversationStatus `json:"status" validate:"required"`
	// Whether the conversation is under legal hold.
	//
	// Exempts the conversation from retention purging and redaction.
	LegalHold constants.LegalHoldStatus `json:"legal_hold" validate:"required"`
	// The case owner, when one is set: a `user` actor (a team member) or a `group` actor (a team).
	Assignee *Actor `json:"assignee" expandable:"true"`
	// The active participants of the conversation.
	Participants *List[ConversationParticipant] `json:"participants" expandable:"true"`
	// The app resource this conversation is anchored to (e.g. an order).
	//
	// `null` when the conversation has no topic anchor.
	Topic *Entity `json:"topic" expandable:"true"`
	// Number of messages the caller has not yet read.
	Unread int64 `json:"unread"`
	// When the most recent message was sent.
	//
	// `null` when the conversation has no messages yet.
	LastMessageAt *time.Time `json:"last_message_at"`
	// The most recent message in the conversation.
	//
	// `null` when the conversation has no messages yet.
	LastMessage *Message `json:"last_message" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A participant (membership) in a conversation.
type ConversationParticipant struct {
	// Participant ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=conversation_participant"`
	// The kind of participant.
	//
	// - `user`: an account user (a teammate).
	// - `agent`: an AI agent.
	// - `system`: the system itself, which posts automated messages.
	// - `customer`: an external customer in a support case.
	Type constants.ParticipantType `json:"type" validate:"required"`
	// The participant's permission level in the conversation.
	//
	// - `owner`: can rename or delete the conversation and manage its members and their roles.
	// - `admin`: can add or remove members and rename the conversation.
	// - `member`: can post, react, mute, and leave.
	// - `viewer`: read-only access.
	Role constants.ParticipantRole `json:"role" validate:"required"`
	// The participant's membership in the conversation.
	//
	// - `active`: currently a member.
	// - `left`: voluntarily left the conversation.
	// - `removed`: removed by an admin.
	// - `hidden`: still a member but has hidden the conversation from their own list.
	Membership constants.ParticipantMembership `json:"membership" validate:"required"`
	// The participant's notification preference for the conversation.
	//
	// - `unmuted`: receives normal notifications.
	// - `muted`: notifications are suppressed (mentions may still pierce the mute).
	Notifications constants.ParticipantNotifications `json:"notifications" validate:"required"`
	// The actor this participant represents: a `user` (account user) or an `agent`.
	//
	// `null` for system participants.
	Actor *Actor `json:"actor"`
	// For agent participants, when the agent is invoked in response to messages.
	//
	// `null` for non-agent participants.
	//
	// - `mention`: only when the agent is @mentioned.
	// - `keyword`: when a message contains one of the agent's trigger keywords.
	// - `always`: on every human message in the conversation.
	AgentTriggerPolicy *constants.AgentTriggerPolicy `json:"agent_trigger_policy"`
	// For agent participants with a keyword/mention policy, the keywords that trigger it.
	AgentTriggerKeywords []string `json:"agent_trigger_keywords"`
	// The participant's read position in the conversation (read receipts): how far they have read.
	ReadCursor ReadCursor `json:"read_cursor"`
}

// A participant's read position in a conversation — the basis for read receipts ("who has seen this").
type ReadCursor struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=read_cursor"`
	// The sequence number of the last message the participant has read in the conversation.
	//
	// A message is "seen" by this participant when its `sequence` is `<=` this value. `0` means they have not read any message in the conversation yet.
	Sequence int64 `json:"sequence"`
	// The id of the last message the participant has read.
	//
	// `null` if they have not read any message yet.
	MessageID *string `json:"message_id"`
	// When the participant last advanced their read cursor.
	//
	// `null` if they have not read any message yet.
	ReadAt *time.Time `json:"read_at"`
}

var SampleConversationParticipant = &ConversationParticipant{
	ID:            SampleConversationParticipantID,
	Object:        constants.ObjectTypeConversationParticipant,
	Type:          constants.ParticipantTypeUser,
	Actor:         NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Jie Yan"), nil),
	Role:          constants.ParticipantRoleMember,
	Membership:    constants.ParticipantMembershipActive,
	Notifications: constants.ParticipantNotificationsUnmuted,
	ReadCursor:    ReadCursor{Object: constants.ObjectTypeReadCursor, Sequence: 12},
}

func (*ConversationParticipant) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConversationParticipant)
}

var SampleConversation = &Conversation{
	ID:        SampleConversationID,
	Object:    constants.ObjectTypeConversation,
	Type:      constants.ConversationTypeDM,
	Audience:  constants.ConversationAudienceInternal,
	Status:    constants.ConversationStatusActive,
	LegalHold: constants.LegalHoldStatusReleased,
	Participants: NewList([]ConversationParticipant{
		{
			ID:            SampleConversationParticipantID,
			Object:        constants.ObjectTypeConversationParticipant,
			Type:          constants.ParticipantTypeUser,
			Actor:         NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Jie Yan"), nil),
			Role:          constants.ParticipantRoleMember,
			Membership:    constants.ParticipantMembershipActive,
			Notifications: constants.ParticipantNotificationsUnmuted,
		},
	}, PageInfo{}),
	Unread:        2,
	LastMessageAt: timeutil.TimestampToTimePtr(sampleUpdatedAtTimestamp),
	LastMessage: &Message{
		ID:         SampleMessageID,
		Object:     constants.ObjectTypeChatMessage,
		Kind:       constants.MessageKindChat,
		Status:     constants.MessageStatusSent,
		Visibility: constants.MessageVisibilityInternal,
		Channel:    constants.MessageChannelMessage,
		Sequence:   42,
		Body:       new("Sounds good — shipping it today."),
		Sender:     NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Jie Yan"), nil),
		CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Conversation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConversation)
}
