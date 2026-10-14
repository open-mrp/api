package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const (
	SampleConversationID            = "cv_w35z4ck68yq7"
	SampleConversationParticipantID = "cvpt_be2h3ul14cts"
)

// A thread of messages between people, devices and agents.
//
// A conversation can be about a record, such as an order, which appears in `related`.
type Conversation struct {
	// Conversation ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=conversation"`
	// What kind of conversation this is, set from who is in it when it's created.
	//
	// - `direct_message`: you and at most one other person.
	// - `group`: anything else, including customer-facing support cases.
	// - `system`: a system channel that delivers automated account alerts.
	Type constants.ConversationType `json:"type" validate:"required"`
	// Whether this is a team-only conversation (`internal`) or a customer-facing case (`customer`).
	//
	// A customer never sees an `internal` conversation, even one that is about them; within a `customer` case they see only the messages that were sent to them, not the team's internal notes on the case.
	Audience constants.ConversationAudience `json:"audience" validate:"required"`
	// The conversation's title.
	//
	// Null until someone names it; clients then show the participants instead.
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
	// Whether the conversation is archived for everyone: `active` or `archived`.
	//
	// Your own state (hidden, muted, how far you've read) is on your participant in `participants`.
	Status constants.ConversationStatus `json:"status" validate:"required"`
	// Whether the conversation is under legal hold.
	//
	// While held, the conversation is exempt from automatic retention purging and from redaction until the hold is released.
	//
	// Null to customer and supplier portal users: a hold is your own legal matter.
	LegalHold *constants.LegalHoldStatus `json:"legal_hold" validate:"required" sensitive:"internal"`
	// The owner of the case: either a `user` actor (an individual team member) or a `group` actor (a team).
	Assignee *Actor `json:"assignee" expandable:"true"`
	// The participants of the conversation.
	//
	// Only current members are listed; anyone who left or was removed is omitted, even though their past messages remain in the thread.
	Participants *List[ConversationParticipant] `json:"participants" expandable:"true"`
	// Records this conversation is about or picks up from.
	Related *ConversationRelated `json:"related"`
	// Messages the caller hasn't read yet.
	UnreadCount int64 `json:"unread_count"`
	// When the most recent message was sent.
	LastMessageAt *time.Time `json:"last_message_at"`
	// The most recent message in the conversation.
	LastMessage *Message `json:"last_message" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Records a conversation is about or picks up from, as lightweight references. Retrieve a record by its ID for full details.
type ConversationRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=conversation_related"`
	// The record the conversation is about, such as an order. Its discussions are listed with the `record_type` and `record_id` filters.
	Record *Record `json:"record" expandable:"true"`
	// The conversation this one picks up from.
	Continues *Record `json:"continues" expandable:"true"`
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
	ID:           SampleConversationID,
	Object:       constants.ObjectTypeConversation,
	Type:         constants.ConversationTypeDM,
	Audience:     constants.ConversationAudienceInternal,
	Status:       constants.ConversationStatusActive,
	LegalHold:    new(constants.LegalHoldStatusReleased),
	Participants: NewList([]ConversationParticipant{*SampleConversationParticipant}, PageInfo{}),
	Related: &ConversationRelated{
		Object: constants.ObjectTypeConversationRelated,
		Record: &Record{ID: SampleSalesOrderID, Object: constants.ObjectTypeRecord, Type: constants.RecordTypeSalesOrder, Number: new("SO-1042"), Status: new("open"), Metadata: map[string]string{}},
	},
	UnreadCount:   2,
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
