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
	// - `send`: delivers the message (immediately, or at `scheduled_at`). This is the default.
	// - `draft`: proposes a customer-reply draft on an external case, held for human approval rather than sent. Requires `channel`.
	Mode field.Optional[constants.MessageSendMode] `json:"mode,omitzero" default:"send"`
	// The channel a draft will be sent over when approved (`mode` = `draft`).
	//
	// - `message`: delivered as an in-conversation chat message.
	// - `email`: delivered as an outbound email from the conversation's bridged inbox.
	Channel field.Optional[constants.MessageChannel] `json:"channel,omitzero"`
	// The internal thread message a draft is composed from, when drafting from a thread (`mode` = `draft`).
	SourceThreadMessageID field.Optional[string] `json:"source_thread_message_id,omitzero"`
	// Client-supplied dedupe key.
	//
	// A resend with the same value returns the original message. Required when sending (`mode` = `send`); ignored for drafts.
	ClientMessageID string `json:"client_message_id,omitzero"`
	// Who the message is addressed to on an external case.
	//
	// - `customer`: sends a customer-visible reply, branded "Customer Service" and delivered by email on an email-bridged case.
	// - `internal`: posts a team-only note that the customer never sees.
	//
	// When omitted, the message is posted as an internal team-only note.
	Audience field.Optional[constants.ConversationAudience] `json:"audience,omitzero" default:"internal"`
	// The email subject for a customer reply on an email-bridged case (`audience` = `customer`).
	Subject field.Optional[string] `json:"subject,omitzero"`
	// Additional email recipients to copy on a customer reply (email channel).
	Cc []string `json:"cc,omitzero"`
	// When set, queue the message for delivery at this future time instead of sending now.
	//
	// The created message has status `scheduled`.
	ScheduledAt field.Optional[time.Time] `json:"scheduled_at,omitzero"`
	// The message this one is a reply to.
	ReplyToMessageID field.Optional[string] `json:"reply_to_message_id,omitzero"`
	// Type of a resource to link in the message, paired with `link_resource_id`.
	LinkResourceType field.Optional[constants.ObjectType] `json:"link_resource_type,omitzero"`
	// ID of a resource to link in the message, paired with `link_resource_type`.
	LinkResourceID field.Optional[string] `json:"link_resource_id,omitzero"`
	// Attachments to include with the message.
	Attachments []MessageAttachmentInput `json:"attachments,omitzero"`
	// Account user ids explicitly @mentioned in the message.
	//
	// A mention delivers a notification even when the recipient has muted the conversation.
	Mentions []string `json:"mentions,omitzero"`
}

// A single attachment supplied when sending a message.
//
// For uploaded kinds (`file`/`image`) supply the `s3_key` returned by the upload-url endpoint; for `link` supply `url`; for `resource` supply `resource_type` and `resource_id`.
type MessageAttachmentInput struct {
	// The kind of attachment.
	Kind constants.MessageAttachmentKind `json:"kind" validate:"required"`
	// The object-storage key from the upload-url response (required for file/image).
	S3Key field.Optional[string] `json:"s3_key,omitzero"`
	// The original filename (file/image).
	Filename field.Optional[string] `json:"filename,omitzero"`
	// The MIME content type (file/image).
	ContentType field.Optional[string] `json:"content_type,omitzero"`
	// The size in bytes (file/image).
	SizeBytes field.Optional[int64] `json:"size_bytes,omitzero"`
	// The external URL (required for link).
	URL field.Optional[string] `json:"url,omitzero"`
	// The linked resource type (required for resource).
	ResourceType field.Optional[string] `json:"resource_type,omitzero"`
	// The linked resource id (required for resource).
	ResourceID field.Optional[string] `json:"resource_id,omitzero"`
}

var sampleSendMessageRequest = &SendMessageRequest{
	ConversationID:  apiresource.SampleConversationID,
	Body:            "Sounds good — shipping it today.",
	ClientMessageID: "client_msg_8c7d2f",
}

func (*SendMessageRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSendMessageRequest)
}

// Posts a message to a conversation.
//
// With `mode` = `send` (the default) the message is delivered — immediately, or queued when `scheduled_at` is set — and the request is idempotent on `client_message_id`. With `mode` = `draft` a customer-reply draft is proposed on an external case: it is held at status `draft` for human approval rather than sent, and `channel` is required.
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
