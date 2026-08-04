package messageep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to post a message to a conversation.
type SendMessageRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// Message body.
	//
	// Required unless the message carries at least one attachment or a resource link.
	Body string `json:"body" validate:"required_without_all=Attachments LinkResourceID"`
	// Whether to deliver the message now or hold it as a customer-reply draft.
	//
	// - `send`: delivers the message, immediately or at `scheduled_at`.
	// - `draft`: proposes a reply to the customer on a customer-facing case and holds it for a teammate to approve before it goes out. Requires `channel`.
	//
	// A draft is built from `body`, `subject`, `channel`, and `source_thread_message_id` only — attachments, mentions, copied recipients, resource links, replies, and scheduling are not carried onto it.
	Mode field.Optional[constants.MessageSendMode] `json:"mode,omitzero" default:"send"`
	// The channel a draft will be sent over once it is approved (`mode` = `draft`).
	//
	// - `message`: appears in the customer's conversation timeline.
	// - `email`: goes out as an email from the inbox the case is bridged to. Falls back to the conversation timeline if the case has no bridged inbox.
	Channel field.Optional[constants.MessageChannel] `json:"channel,omitzero"`
	// The internal thread message a draft is composed from, when drafting from a thread (`mode` = `draft`).
	SourceThreadMessageID field.Optional[string] `json:"source_thread_message_id,omitzero"`
	// Client-supplied dedupe key.
	//
	// Repeating an immediate send with the same value returns the message created by the first request instead of posting a second one, so a retry after a network failure is safe. Required when sending (`mode` = `send`); ignored for drafts.
	ClientMessageID string `json:"client_message_id,omitzero"`
	// Who the message is addressed to on a customer-facing case.
	//
	// - `customer`: a reply the customer sees, shown to them as coming from "Customer Service" and delivered as email when the case is bridged to an inbox.
	// - `internal`: a team-only note the customer never sees.
	//
	// Messages are team-only unless you ask for `customer`, so an internal note can never leak by omission. Asking for `customer` on a conversation that has no customer is rejected.
	//
	// On a case bridged to an email inbox, a customer reply goes out as mail carrying only the body, subject, and copied recipients — attachments, mentions, resource links, and replies are dropped.
	Audience field.Optional[constants.ConversationAudience] `json:"audience,omitzero" default:"internal"`
	// The subject line for a customer reply sent by email.
	//
	// When omitted, the reply goes out as "Re:" the case title.
	Subject field.Optional[string] `json:"subject,omitzero"`
	// Additional email addresses to copy on a customer reply sent by email.
	Cc []string `json:"cc,omitzero"`
	// When set, hold the message and deliver it at this future time instead of sending it now.
	//
	// Only the body is carried into a scheduled send — attachments, mentions, copied recipients, resource links, replies, and audience are dropped, and it is delivered as an ordinary team-visible message. If you are no longer an active participant when it comes due, it is canceled instead of sent.
	ScheduledAt field.Optional[time.Time] `json:"scheduled_at,omitzero"`
	// The message this one is a reply to.
	ReplyToMessageID field.Optional[string] `json:"reply_to_message_id,omitzero"`
	// Type of a resource to link in the message, paired with `link_resource_id`.
	//
	// Linking a record lets clients render the message as a reference to it. A link counts in place of text, so a message may consist of nothing but the link.
	LinkResourceType field.Optional[constants.ObjectType] `json:"link_resource_type,omitzero"`
	// ID of a resource to link in the message, paired with `link_resource_type`.
	LinkResourceID field.Optional[string] `json:"link_resource_id,omitzero"`
	// Attachments to include with the message.
	Attachments []MessageAttachmentInput `json:"attachments,omitzero"`
	// Account user ids explicitly @mentioned in the message.
	//
	// A mention notifies the person even when they have muted the conversation.
	Mentions []string `json:"mentions,omitzero"`
}

// A single attachment supplied when sending a message.
//
// For an uploaded file or image, supply the `s3_key` you uploaded to; for a link, supply `url`; for a resource reference, supply `resource_type` and `resource_id`.
type MessageAttachmentInput struct {
	// What is being attached.
	//
	// - `file`: a document you uploaded to object storage first.
	// - `image`: an uploaded image, rendered inline in the conversation.
	// - `link`: an external web address, with nothing stored on our side.
	// - `resource`: a reference to an in-app record, such as an order.
	Kind constants.MessageAttachmentKind `json:"kind" validate:"required"`
	// The key you uploaded the file to, taken from the upload-url response (file and image).
	//
	// The key must be one minted for this conversation and the file must already be uploaded, otherwise the send is rejected.
	S3Key field.Optional[string] `json:"s3_key,omitzero"`
	// The filename to display for the attachment (file and image).
	Filename field.Optional[string] `json:"filename,omitzero"`
	// The MIME content type of the uploaded file (file and image).
	ContentType field.Optional[string] `json:"content_type,omitzero"`
	// The size of the uploaded file in bytes (file and image).
	SizeBytes field.Optional[int64] `json:"size_bytes,omitzero"`
	// The web address being shared (link).
	URL field.Optional[string] `json:"url,omitzero"`
	// The type of the record being referenced, paired with `resource_id` (resource).
	ResourceType field.Optional[string] `json:"resource_type,omitzero"`
	// The id of the record being referenced, paired with `resource_type` (resource).
	ResourceID field.Optional[string] `json:"resource_id,omitzero"`
}

var sampleSendMessageSubject = "Re: Order #1042"

var sampleSendMessageRequest = &SendMessageRequest{
	ConversationID:        apiresource.SampleConversationID,
	Body:                  "Sounds good — shipping it today.",
	Mode:                  field.Some(constants.MessageSendModeSend),
	Channel:               field.Some(constants.MessageChannelEmail),
	SourceThreadMessageID: field.Some(apiresource.SampleMessageID),
	ClientMessageID:       "client_msg_8c7d2f",
	Audience:              field.Some(constants.ConversationAudienceCustomer),
	Subject:               field.Some(sampleSendMessageSubject),
	Cc:                    []string{"ap@acme.com"},
	ScheduledAt:           field.Some(time.Date(2026, 5, 10, 15, 0, 0, 0, time.UTC)),
	ReplyToMessageID:      field.Some(apiresource.SampleMessageID),
	LinkResourceType:      field.Some(constants.ObjectTypeSalesOrder),
	LinkResourceID:        field.Some(apiresource.SampleSalesOrderID),
	Attachments: []MessageAttachmentInput{{
		Kind:        constants.MessageAttachmentKindFile,
		S3Key:       field.Some("uploads/acme/quote.pdf"),
		Filename:    field.Some("quote.pdf"),
		ContentType: field.Some("application/pdf"),
		SizeBytes:   field.Some(int64(20480)),
	}},
	Mentions: []string{apiresource.SampleAccountUserID},
}

func (*SendMessageRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSendMessageRequest)
}

// Posts a message to a conversation.
//
// With `mode` = `send` the message is delivered — immediately, or queued when `scheduled_at` is set — and a retry of an immediate send with the same `client_message_id` returns the original message rather than posting it twice. With `mode` = `draft` the message is proposed as a reply to the customer and held for a teammate to approve instead of being sent, and `channel` is required.
//
// Sending requires you to be an active participant allowed to post: view-only participants cannot post, and in a direct message neither side of a block can. On a customer-facing case, replying to the customer moves the case to waiting on the customer, and proposing a draft moves it to awaiting approval.
type SendMessageEndpoint struct{}

func (e *SendMessageEndpoint) Materialize() *apiendpoint.APIEndpoint[*SendMessageRequest, *apiresource.Message] {
	return (&apiendpoint.APIEndpoint[*SendMessageRequest, *apiresource.Message]{
		Title:               "Send Message",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/messages",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeChatMessage,
		IncludeConfig:       messageIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SendMessageRequest) (*apiresource.Message, *apierror.APIError) {
			return svc.(MessageSvc).SendMessage
		},
	})
}
