package domain

import (
	"encoding/json"
	"time"
)

// Conversation is a messaging container (dm/group/system/...). Unread is caller-scoped and is only populated by list/get paths that join the caller's participant row.
type Conversation struct {
	ID        string
	AccountID string
	Type      string `audit:"type"`
	// Audience is the direction of the conversation: "internal" (team-only; the customer is never a participant) or "customer" (a customer-facing external case). Drives the support inbox and the customer-visible read filtering.
	Audience string  `audit:"audience"`
	Title    *string `audit:"title"`
	// GroupID is the reusable roster (messaging_group.id) this conversation was seeded from, if any. Provenance only — members were snapshotted into participants at create time and are not driven by the group thereafter.
	GroupID           *string `audit:"group_id"`
	TopicResourceType *string
	TopicResourceID   *string
	NextSequence      int64
	LastMessageID     *string
	LastMessageAt     *time.Time
	IsArchived        bool `audit:"is_archived"`
	LegalHold         bool `audit:"legal_hold"`
	// External customer-service case triage (audience=customer only; nil/empty on internal).
	WorkflowStatus *string `audit:"workflow_status"`
	// Assignee is the single polymorphic case owner — an account_user or an account_group (type + id).
	AssigneeResourceType *string `audit:"assignee_resource_type"`
	AssigneeResourceID   *string `audit:"assignee_resource_id"`
	// EmailInboxID is set when the case is bridged to an email inbox (outbound replies route through it).
	EmailInboxID *string
	Metadata     json.RawMessage
	CreatedAt    time.Time
	UpdatedAt    time.Time
	// Caller-scoped fields (populated on list/get for the requesting participant).
	Unread       int64
	Hidden       bool
	Participants []*ConversationParticipant
	// LastMessage is the conversation's most recent message, hydrated on list for inbox previews (nil when the conversation has no messages). Attachments/sender are not resolved on this preview copy — it carries body/preview/deleted/timestamp only.
	LastMessage *Message
}

// ConversationParticipant is a membership row plus the participant's per-user state.
type ConversationParticipant struct {
	ID                string
	ConversationID    string `audit:"conversation_id"`
	AccountID         string
	ParticipantType   string  `audit:"participant_type"`
	AccountUserID     *string `audit:"account_user_id"`
	AgentConfigID     *string `audit:"agent_config_id"`
	Role              string  `audit:"role"`
	Membership        string  `audit:"membership"`
	Notifications     string  `audit:"notifications"`
	LastReadSequence  int64
	LastReadMessageID *string
	LastReadAt        *time.Time
	HiddenAt          *time.Time
	JoinedAt          time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	// Agent-participant trigger config (participant_type=agent only).
	AgentTriggerPolicy   *string  `audit:"agent_trigger_policy"`
	AgentTriggerKeywords []string `audit:"agent_trigger_keywords"`
	// RelationAccountID is the customer/supplier account for participant_type=customer.
	RelationAccountID *string
	// AccountUserDisplayName (not persisted) is the resolved display name of a customer participant's account_user (cross-account contact lookup for staff viewers).
	AccountUserDisplayName *string
}

// SupportRoute is the group conversation designated to handle a relationship's inbound support.
// RelationAccountID is the scope: "" = the account-level default for any customer; a concrete account id is a per-relation override. The routed group conversation's participants are seated as the deterministic recipients on the customer's support thread.
type SupportRoute struct {
	ID                  string
	AccountID           string
	RelationAccountID   string `audit:"relation_account_id"`
	GroupConversationID string `audit:"group_conversation_id"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// AddAgentParticipantInput is the validated input for adding an agent to a conversation.
type AddAgentParticipantInput struct {
	ConversationID  string
	AgentConfigID   string
	TriggerPolicy   string
	TriggerKeywords []string
}

// Message is a single message row. SenderAccountUserID is resolved (not persisted) from the sender participant for presentation.
type Message struct {
	ID             string
	ConversationID string
	AccountID      string
	// Sequence is the per-conversation timeline position. It is 0 (unassigned) for unsent rows
	// (status draft/scheduled), which hold a NULL sequence in the DB until promoted to "sent".
	Sequence int64
	Kind     string
	// Status is the lifecycle state: "sent" (timeline), "scheduled" (future delivery), "draft"
	// (customer-reply draft), or a terminal "canceled"/"rejected"/"failed"/"superseded". Only "sent"
	// rows appear in the conversation timeline. See constants.MessageStatus.
	Status string
	// Visibility is the message's audience inside the conversation: "internal" (team-only note),
	// "external" (sent to/from an external party), or "system" (event shown to both). Customer read paths hard-filter to external+system so an internal note is never serialized into a customer payload.
	Visibility          string
	SenderParticipantID *string
	ClientMessageID     *string
	Body                *string
	// Subject is the customer-reply draft email subject (email channel); nil for ordinary messages.
	Subject *string
	// Channel is how the message was delivered (or, for a draft, how it will be on approve): "message" | "email".
	Channel *string
	// SourceThreadMessageID is the internal thread message a draft was composed from (draft provenance).
	SourceThreadMessageID *string
	// ApprovedByAccountUserID is the account_user who approved a draft on send.
	ApprovedByAccountUserID *string
	// ScheduledFor is the future delivery time for a status=scheduled message (UTC); nil otherwise.
	ScheduledFor *time.Time
	// ScheduledAttempts / LastError / LockedAt / LockOwner are scheduler bookkeeping for status=scheduled rows.
	ScheduledAttempts int32
	LastError         *string
	LockedAt          *time.Time
	LockOwner         *string
	Preview           *string
	EventType         *string
	LinkResourceType  *string
	LinkResourceID    *string
	// AgentRunID is the agent-service run that produced this message (string ref across the DB boundary; null for human/system messages). Bridges a chat reply to its run for "expand run".
	AgentRunID       *string
	ReplyToMessageID *string
	// StreamingState is "streaming" while an agent reply is still being generated (its body fills in via realtime patches) and "complete" once finalized. Empty/"complete" for ordinary messages.
	StreamingState *string
	EditedAt       *time.Time
	DeletedAt      *time.Time
	Metadata       json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
	// Resolved sender (not persisted): the author's account_user id, for presentation. Cleared for a customer-relation viewer when the message is collapsed to the branded "Customer Service" alias.
	SenderAccountUserID *string
	// Resolved sender (not persisted): the author's agent_config id, set when the sender participant is an agent. The author is a polymorphic actor — exactly one of SenderAccountUserID /
	// SenderAgentConfigID is set.
	SenderAgentConfigID *string
	// SenderAgentName (not persisted) is the agent definition's display name when the author is an agent; used to title chat bell notifications.
	SenderAgentName *string
	// SenderAlias (not persisted) is the branded "shown as" name presented to a customer-relation viewer in place of the real staff/agent author (read-time anonymization on external cases). When set, the real author fields above are cleared.
	SenderAlias *string
	// SenderDisplayName (not persisted) is the resolved contact display name for a customer-participant author when the viewer is internal staff (the account_user lives in the customer's account and cannot be hydrated via the vendor-scoped account_user batch loader at the gateway).
	SenderDisplayName *string
	// Resolved attachments (not persisted on the message row); populated on read with presigned URLs.
	Attachments []*MessageAttachment
	// MentionAccountUserIDs are the @mentioned account_users (transient, carried from the send input to the notification fanout so a mention can pierce the recipient's mute).
	MentionAccountUserIDs []string
}

// CreateConversationInput is the validated input for creating a conversation.
type CreateConversationInput struct {
	Type string
	// Audience is "internal" (default) or "customer"; empty is treated as "internal".
	Audience          string
	Title             *string
	TopicResourceType *string
	TopicResourceID   *string
	// GroupID, when set, seeds the conversation from a reusable roster: the group's members are snapshotted into participants (in addition to ParticipantAccountUserIDs) and the conversation records this group_id as provenance.
	GroupID                   *string
	ParticipantAccountUserIDs []string
}

// ConversationListFilter parameterizes the caller's conversation list.
type ConversationListFilter struct {
	AccountID           string
	AccountUserID       string
	Type                *string
	Status              string
	Limit               int32
	CursorLastMessageAt *time.Time
	CursorID            *string
}

// ConversationPage is one page of the caller's conversations plus the next opaque cursor.
type ConversationPage struct {
	Conversations []*Conversation
	NextCursor    *string
	HasNextPage   bool
}

// SendMessageInput is the validated input for sending a chat message.
type SendMessageInput struct {
	ConversationID string
	// Visibility is the requested audience for this message. Empty defaults to a safe value resolved by the service (internal for external cases). On an external case, "external" requires the send-customer-reply permission; on an internal conversation it is always forced to "internal".
	Visibility       string
	Body             *string
	ClientMessageID  string
	ReplyToMessageID *string
	LinkResourceType *string
	LinkResourceID   *string
	// Subject is used only for a customer reply on an email-bridged case (Visibility=external); the
	// outbound email subject. Defaults to the case title when empty. Ignored for portal/internal sends.
	Subject *string
	// Cc lists additional email recipients for a customer reply on an email-bridged case. Email channel only.
	Cc []string
	// Attachments are optional files/images/links/resources to attach to the message.
	Attachments []AttachmentInput
	// MentionAccountUserIDs are the account_users explicitly @mentioned in the message. A mention delivers a chat.mention bell even when the recipient has muted the conversation.
	MentionAccountUserIDs []string
}

// AgentReplyInput is a chat reply produced by an agent run, posted into a conversation as the agent participant (resolved from ConversationID + AgentConfigID) and linked to AgentRunID.
type AgentReplyInput struct {
	AccountID       string
	ConversationID  string
	AgentConfigID   string
	AgentName       string
	AgentRunID      string
	Body            string
	ClientMessageID string
	// MessageID is the agent-owned row id for a streaming reply (shared by start/patch/complete). Empty for the legacy single-shot path, where the service generates the id.
	MessageID string
	// ReplyToMessageID threads the reply under the message that triggered the run (mention/keyword).
	ReplyToMessageID string
	// Failed marks a reply that resolves an errored run. ErrorCode is the machine-readable api-error code (e.g. "agent_spending_cap_reached"); both are recorded on the message so the client can flag the failure and react (e.g. prompt to raise the spending limit).
	Failed    bool
	ErrorCode string
}

// AgentReplyPatchInput is a best-effort partial-body update for an in-flight streaming agent reply, addressing the row by MessageID. Body is the full accumulated answer so far.
type AgentReplyPatchInput struct {
	AccountID      string
	ConversationID string
	MessageID      string
	Body           string
}

// SystemEventInput is a senderless timeline event posted into a conversation (the Body is the full sentence). EventType is a discriminator stored on the row; ClientMessageID makes it idempotent.
type SystemEventInput struct {
	AccountID       string
	ConversationID  string
	EventType       string
	Body            string
	ClientMessageID string
}

// MessageListFilter parameterizes a conversation's message history query.
type MessageListFilter struct {
	ConversationID string
	Limit          int32
	BeforeSequence *int64
	AfterSequence  *int64
	// IncludeInternal controls whether internal-only messages are returned. False for customer-relation viewers (they see only external+system messages); true for internal staff.
	IncludeInternal bool
}

// MessagePage is one page of a conversation's messages plus the next opaque cursor (older).
type MessagePage struct {
	Messages    []*Message
	NextCursor  *string
	HasNextPage bool
}

// MessageBlock records one account_user blocking another within an account.
type MessageBlock struct {
	ID                   string
	AccountID            string
	BlockerAccountUserID string `audit:"blocker_account_user_id"`
	BlockedAccountUserID string `audit:"blocked_account_user_id"`
	CreatedAt            time.Time
}

// Messaging contact types: "user" for an internal account_user, "support" for the single customer-facing support contact.
const (
	MessagingContactTypeUser    = "user"
	MessagingContactTypeSupport = "support"
)

// MessagingContact is one messageable target in the messaging directory. Type is "user" for an internal account_user (AccountUserID populated) or "support" for the customer-facing support contact (AccountUserID empty).
type MessagingContact struct {
	Type          string
	AccountUserID string
	Name          string
}

// CreateReplyDraftInput is the validated input for proposing a customer-reply draft (a message row at
// status=draft). On approve-send it is promoted in place to a customer-visible sent message.
type CreateReplyDraftInput struct {
	ConversationID        string
	Channel               string
	Subject               *string
	Body                  string
	SourceThreadMessageID *string
	// AgentRunID is set when an agent produced the draft (cross-DB string ref); empty for human drafts.
	AgentRunID string
}

// ConversationLink is a secondary business-record link on a conversation (order, invoice, …).
type ConversationLink struct {
	ID                     string
	AccountID              string
	ConversationID         string `audit:"conversation_id"`
	ResourceType           string `audit:"resource_type"`
	ResourceID             string `audit:"resource_id"`
	CreatedByParticipantID *string
	CreatedAt              time.Time
}

// SupportInboxInput parameterizes the customer-service inbox list (opaque cursor).
type SupportInboxInput struct {
	WorkflowStatus     *string
	AssigneeResourceID *string
	Unassigned         bool
	IncludeArchived    bool
	Cursor             *string
	Limit              int32
}

// SupportInboxFilter parameterizes the customer-service inbox (external cases for triage).
type SupportInboxFilter struct {
	AccountID          string
	WorkflowStatus     *string
	AssigneeResourceID *string
	// Unassigned, when true, restricts to cases with no assignee.
	Unassigned          bool
	IncludeArchived     bool
	Limit               int32
	CursorLastMessageAt *time.Time
	CursorID            *string
}

// MessageReport is a minimal abuse report filed by a participant against a conversation (optionally a specific message). MessageID is empty when the whole conversation is reported.
type MessageReport struct {
	ID                    string
	AccountID             string
	ConversationID        string
	MessageID             string
	ReporterAccountUserID string
	Reason                string
	CreatedAt             time.Time
}
