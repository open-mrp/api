package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMessageID = "mg_01h9z8q1w2e3r4t5y6u7i8mg"

// A chat message within a conversation.
type Message struct {
	// Message ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=chat_message"`
	// The kind of message.
	//
	// - `chat`: a user-authored chat message.
	// - `system_event`: a system-generated event message.
	// - `agent`: a message authored by an AI agent participant.
	// - `scheduled`: a message materialized from a scheduled send.
	// - `alert`: a system or producer alert rendered as a message.
	// - `email`: an inbound email materialized into the conversation by the email bridge.
	Kind constants.MessageKind `json:"kind" validate:"required"`
	// The lifecycle state of the message.
	//
	// - `draft`: an editable customer-reply draft awaiting approval; not in the timeline.
	// - `scheduled`: queued for delivery at a future time; not yet in the timeline.
	// - `sent`: a delivered timeline message; only `sent` messages carry a `sequence`.
	// - `canceled`: a scheduled message canceled before delivery.
	// - `rejected`: a draft discarded without sending.
	// - `failed`: a scheduled message that exhausted delivery attempts.
	// - `superseded`: a draft replaced by a newer one for the same source thread.
	Status constants.MessageStatus `json:"status" validate:"required"`
	// Who can see this message.
	//
	// - `internal`: a team-only note.
	// - `external`: sent to or received from an external party (e.g. the customer on a support case).
	// - `system`: an event shown to both the team and the customer.
	//
	// On a customer-facing conversation, customer payloads only ever carry `external` and `system` messages.
	Visibility constants.MessageVisibility `json:"visibility" validate:"required"`
	// The conversation this message belongs to.
	Conversation *Conversation `json:"conversation" expandable:"true"`
	// Monotonic per-conversation ordering sequence.
	Sequence int64 `json:"sequence"`
	// Message body.
	//
	// `null` for templated or deleted messages.
	Body *string `json:"body"`
	// The email subject of a customer-reply `draft` on an email-bridged case.
	Subject *string `json:"subject"`
	// The actor that sent the message, as displayed. When the message was posted under a sender identity (a persona / group), this is that persona; otherwise it is the authoring user.
	//
	// `null` for pure system messages.
	Sender *Actor `json:"sender" expandable:"true"`
	// The underlying account user who authored the message.
	//
	// `null` for system messages, or when the message was posted under an anonymizing sender identity and the caller is not entitled to see the real author.
	Author *Actor `json:"author" expandable:"true"`
	// Files, images, links, or resources attached to the message.
	Attachments *List[MessageAttachment] `json:"attachments" expandable:"true"`
	// The message this one replies to.
	ReplyTo *Message `json:"reply_to" expandable:"true"`
	// The app resource this message links to.
	Resource *Entity `json:"resource" expandable:"true"`
	// How the message was delivered (or, for a draft, how it will be on approve).
	//
	// - `message`: delivered as an in-conversation chat message.
	// - `email`: delivered as email through the conversation's bridged inbox.
	Channel constants.MessageChannel `json:"channel" validate:"required"`
	// When a `scheduled` message is due to be delivered.
	ScheduledAt *time.Time `json:"scheduled_at"`
	// The agent run that produced this message, for deep-linking from an agent reply to its run.
	//
	// `null` for messages not produced by an agent.
	AgentRun *AgentRun `json:"agent_run" expandable:"true"`
	// The streaming state of a reply.
	//
	// `streaming` while the body is still being generated (it fills in via realtime updates); `complete` once finalized.
	//
	// `null` for ordinary messages.
	StreamingState *string `json:"streaming_state"`
	// The client-supplied dedupe key echoed back for optimistic-UI reconciliation.
	//
	// `null` for server-generated messages.
	ClientMessageID *string `json:"client_message_id"`
	// When the message was last edited.
	EditedAt *time.Time `json:"edited_at"`
	// When the message was deleted (tombstone).
	DeletedAt *time.Time `json:"deleted_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleMessage = &Message{
	ID:           SampleMessageID,
	Object:       constants.ObjectTypeChatMessage,
	Kind:         constants.MessageKindChat,
	Status:       constants.MessageStatusSent,
	Visibility:   constants.MessageVisibilityInternal,
	Channel:      constants.MessageChannelMessage,
	Conversation: SampleConversation,
	Sequence:     42,
	Body:         new("Sounds good — shipping it today."),
	Sender:       NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Jie Yan"), nil),
	Author:       NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Jie Yan"), nil),
	Attachments:  NewList([]MessageAttachment{*SampleMessageAttachment}, PageInfo{}),
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Message) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMessage)
}
