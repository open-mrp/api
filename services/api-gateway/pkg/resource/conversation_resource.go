package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const (
	SampleConversationID            = "cv_w35z4ck68yq7"
	SampleConversationParticipantID = "cvpt_be2h3ul14cts"
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
	//
	// A customer never sees an `internal` conversation, even one that is about them; within a `customer` case they see only the messages that were sent to them, not the team's internal notes on the case.
	Audience constants.ConversationAudience `json:"audience" validate:"required"`
	// The display title of a group conversation.
	//
	// Direct messages carry no stored title; clients derive one from the participants.
	Title *string `json:"title"`
	// The triage lane of a customer-facing case.
	//
	// Only conversations with a `customer` audience have a triage lane. It drives the support inbox and is independent of `status`, which is about visibility rather than progress.
	//
	// - `new`: opened but not yet triaged.
	// - `open`: actively being worked.
	// - `waiting_internal`: blocked on the internal team.
	// - `waiting_external`: blocked on an external reply.
	// - `needs_approval`: a drafted reply is awaiting human approval.
	// - `resolved`: closed out.
	WorkflowStatus *constants.ConversationWorkflowStatus `json:"workflow_status"`
	// The reusable roster this conversation was started from.
	//
	// This is provenance only: the roster's members were copied into the conversation when it was created, so later edits to the roster never add or remove participants here, and deleting the roster only clears this reference.
	Group *MessagingGroup `json:"group" expandable:"true"`
	// The conversation's state from the caller's point of view.
	//
	// - `active`: a normal, visible conversation.
	// - `archived`: archived for the whole account.
	// - `hidden`: the caller dismissed the conversation from their own list while everyone else still sees it, which takes precedence over an account-level archive.
	Status constants.ConversationStatus `json:"status" validate:"required"`
	// Whether the conversation is under legal hold.
	//
	// While held, the conversation is exempt from automatic retention purging and from redaction until the hold is released.
	LegalHold constants.LegalHoldStatus `json:"legal_hold" validate:"required"`
	// The owner of the case: either a `user` actor (an individual team member) or a `group` actor (a team).
	Assignee *Actor `json:"assignee" expandable:"true"`
	// The participants of the conversation.
	//
	// Only current members are listed; anyone who left or was removed is omitted, even though their past messages remain in the thread.
	Participants *List[ConversationParticipant] `json:"participants" expandable:"true"`
	// The app record this conversation is anchored to, such as a sales order.
	//
	// Anchored conversations surface as the discussion thread on that record.
	Topic *Entity `json:"topic" expandable:"true"`
	// Number of messages the caller has not yet read.
	Unread int64 `json:"unread"`
	// When the most recent message was sent.
	LastMessageAt *time.Time `json:"last_message_at"`
	// The most recent message in the conversation.
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
	//
	// Membership records are kept rather than deleted, so re-adding someone who left or was removed reactivates their original record and their earlier messages stay attributed to them.
	Membership constants.ParticipantMembership `json:"membership" validate:"required"`
	// The participant's notification preference for the conversation.
	//
	// - `unmuted`: receives notifications for new messages.
	// - `muted`: new-message notifications are suppressed, though a direct @mention still raises an in-app alert (never an email), and the conversation still counts toward the unread total.
	Notifications constants.ParticipantNotifications `json:"notifications" validate:"required"`
	// The user or agent behind this participant.
	//
	// A customer participant resolves to the `user` actor of the person who opened the case; the `system` participant that posts automated messages has no actor.
	Actor *Actor `json:"actor"`
	// For agent participants, when the agent is invoked in response to messages.
	//
	// - `mention`: only when the agent is @mentioned.
	// - `keyword`: when a message contains one of the agent's trigger keywords.
	// - `always`: on every human message in the conversation.
	AgentTriggerPolicy *constants.AgentTriggerPolicy `json:"agent_trigger_policy"`
	// For agent participants with a keyword or mention policy, the keywords that trigger it.
	//
	// Matching is case-insensitive and looks anywhere in the message body: under `keyword` the bare word is matched, under `mention` it must appear as `@keyword`. Replying directly to one of the agent's own messages always reaches it, so an agent with no keywords still answers replies but nothing else.
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
	MessageID *string `json:"message_id"`
	// When the participant last advanced their read cursor.
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
	ID:            SampleConversationID,
	Object:        constants.ObjectTypeConversation,
	Type:          constants.ConversationTypeDM,
	Audience:      constants.ConversationAudienceInternal,
	Status:        constants.ConversationStatusActive,
	LegalHold:     constants.LegalHoldStatusReleased,
	Participants:  NewList([]ConversationParticipant{*SampleConversationParticipant}, PageInfo{}),
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
