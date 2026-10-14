package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleMessageID = "mg_fdny8633ebgw"

// A chat message within a conversation.
//
// One resource covers every stage of a message's life: a delivered timeline message, a message queued for a future send, and a customer-reply draft awaiting approval. Read `status` to tell them apart.
type Message struct {
	// Message ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=chat_message"`
	// What this message represents.
	//
	// - `chat`: written by a person.
	// - `system_event`: a record of something that happened in the conversation, such as someone joining or a record being linked.
	// - `agent`: written by an AI agent taking part in the conversation.
	// - `scheduled`: came from a send queued ahead of time.
	// - `alert`: an automated alert surfaced in the conversation.
	// - `email`: a message carried over the case's bridged email thread, either one that arrived from the customer or a reply sent back out to them.
	Kind constants.MessageKind `json:"kind" validate:"required"`
	// Where the message stands in its life.
	//
	// - `draft`: a proposed reply to the customer, still editable and waiting for approval before anyone outside sees it.
	// - `scheduled`: queued to go out at a future time.
	// - `sent`: delivered, and part of the conversation everyone reads.
	// - `failed`: a scheduled message that could not be delivered, or an agent reply whose work failed; `error` says why.
	// - `deleted`: deleted or redacted. It keeps its place in the conversation, but its `body` and `attachments` are null.
	// - `canceled`: a scheduled message stopped before it went out.
	// - `rejected`: a draft discarded instead of being sent.
	// - `superseded`: a draft replaced by a newer one for the same thread.
	//
	// Only `sent`, `deleted` and failed agent replies occupy a place in the conversation; the others are records of messages that never reached it.
	Status constants.MessageStatus `json:"status" validate:"required"`
	// Who can see this message.
	//
	// - `internal`: a note only your team can see.
	// - `external`: sent to or received from an outside party, such as the customer on a support case, and part of the official record of that exchange.
	// - `system`: an event both your team and the customer see.
	//
	// A customer reading their own case is never served `internal` messages.
	Visibility constants.MessageVisibility `json:"visibility" validate:"required"`
	// The conversation this message belongs to.
	Conversation *Conversation `json:"conversation" expandable:"true"`
	// The message's position in the conversation timeline, counting up from the first message.
	//
	// A sequence is assigned only when a message is delivered, so a draft or a not-yet-sent scheduled message reports `0`. Listing a conversation's messages pages backwards through this ordering.
	Sequence int64 `json:"sequence"`
	// Message body.
	//
	// Null on a message made up of nothing but attachments or a linked record, and on a deleted message.
	Body *string `json:"body"`
	// The email subject line.
	//
	// On an email-bridged case, this is the subject of the inbound email, or the subject a customer reply is sent out with.
	Subject *string `json:"subject"`
	// The party the message is displayed as coming from.
	//
	// On a customer-facing case the customer sees every reply from your side as a single branded "Customer Service" party rather than the individual person or agent behind it, and an inbound email is shown as the outside address it arrived from. Everywhere else this is the user or agent that wrote the message. Pure system messages have no sender.
	Sender *Actor `json:"sender" expandable:"true"`
	// The user or agent that actually wrote the message.
	//
	// Absent on system messages, and on a vendor-side reply read by a customer — the real author behind the branded "Customer Service" party is never revealed to them.
	Author *Actor `json:"author" expandable:"true"`
	// Files, images, links, or records attached to the message. Null on a deleted message.
	Attachments *List[MessageAttachment] `json:"attachments" expandable:"true"`
	// The message this one replies to.
	ReplyTo *Message `json:"reply_to" expandable:"true"`
	// The record this message links to.
	Related *MessageRelated `json:"related"`
	// How the message reached its audience, or how a draft will be sent once it is approved.
	//
	// - `message`: appears in the conversation itself.
	// - `email`: goes out as email on the thread of the inbox the case is bridged to.
	Channel constants.MessageChannel `json:"channel" validate:"required"`
	// When a message queued for a future send is due to go out.
	ScheduledAt *time.Time `json:"scheduled_at"`
	// The agent run that produced this message, for deep-linking from an agent reply to its run.
	AgentRun *AgentRun `json:"agent_run" expandable:"true"`
	// The streaming state of an agent reply.
	//
	// `streaming` means the body is still being generated and keeps growing as realtime updates arrive; `complete` means it is final.
	StreamingState *constants.MessageStreamingState `json:"streaming_state"`
	// The dedupe key the client supplied when sending, echoed back so an optimistic local copy can be matched to the stored message.
	ClientMessageID *string `json:"client_message_id"`
	// When the message was last edited.
	EditedAt *time.Time `json:"edited_at"`
	// When the message was deleted.
	//
	// A deleted message keeps its place in the timeline with its content cleared, so surrounding ordering and replies stay intact.
	DeletedAt *time.Time `json:"deleted_at"`
	// Why the message failed, when its `status` is `failed` because an agent's work failed; otherwise null.
	Error *MessageError `json:"error"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// The record a message links to, as a lightweight reference. Retrieve the record by its ID for full details.
type MessageRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=chat_message_related"`
	// The record the message links to, such as the order it is about.
	Record *Record `json:"record" expandable:"true"`
}

// Why a message failed.
type MessageError struct {
	// Machine-readable reason, such as `agent_spending_cap_reached`, a cue to offer raising the agent spending limit.
	Code apierror.Code `json:"code" validate:"required"`
	// Human-readable explanation, the same text as the message `body`.
	Message string `json:"message" validate:"required"`
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
	Related: &MessageRelated{
		Object: constants.ObjectTypeChatMessageRelated,
		Record: &Record{ID: SampleSalesOrderID, Object: constants.ObjectTypeRecord, Type: constants.RecordTypeSalesOrder, Number: new("SO-1042"), Status: new("open"), Metadata: map[string]string{}},
	},
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Message) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMessage)
}
