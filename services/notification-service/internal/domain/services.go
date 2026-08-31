package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// ConversationSvc is the 1:1/group chat surface (Phase 2: DM). Reads are participant-scoped; SendMessage allocates a per-conversation sequence and fans out realtime pushes.
type ConversationSvc interface {
	// CreateConversation creates a DM (deduped by participant pair) or other conversation type.
	CreateConversation(ctx context.Context, input CreateConversationInput) (*Conversation, *apierror.APIError)
	// ListConversations returns the caller's conversations (most-recently-active first).
	ListConversations(ctx context.Context, input ListConversationsInput) (*ConversationPage, *apierror.APIError)
	// GetConversation returns one conversation (with participants) the caller belongs to.
	GetConversation(ctx context.Context, id string) (*Conversation, *apierror.APIError)
	// BatchGetConversations returns the requested conversations the caller belongs to. Conversations the caller cannot access are silently omitted (used for ?include= expansion at the gateway).
	BatchGetConversations(ctx context.Context, ids []string) ([]*Conversation, *apierror.APIError)
	// BatchGetMessages returns the requested messages from conversations the caller belongs to.
	// Messages the caller cannot access are silently omitted (used for ?include= expansion).
	BatchGetMessages(ctx context.Context, ids []string) ([]*Message, *apierror.APIError)
	// ContactSupport returns (creating if needed) the calling customer's support conversation in the vendor account. Customer relation actors only; deduped per customer account. Refuses to open a new thread when the vendor has no support route configured (existing threads still resolve).
	ContactSupport(ctx context.Context) (*Conversation, *apierror.APIError)
	// SupportAvailability reports whether the calling customer can contact support (a configured route with at least one recipient). Customer relation actors only.
	SupportAvailability(ctx context.Context) (bool, *apierror.APIError)
	// SetSupportRoute designates the group conversation handling a relationship's support (scope: ""= account default, or a relation account id override). The target must be a group conversation.
	SetSupportRoute(ctx context.Context, relationAccountID, groupConversationID string) (*SupportRoute, *apierror.APIError)
	// ClearSupportRoute removes the route for a scope in the caller's account.
	ClearSupportRoute(ctx context.Context, relationAccountID string) *apierror.APIError
	// GetSupportRoute returns the route for an exact scope in the caller's account.
	GetSupportRoute(ctx context.Context, relationAccountID string) (*SupportRoute, *apierror.APIError)
	// SendMessage posts a message (idempotent on client_message_id) and fans out realtime.
	SendMessage(ctx context.Context, input SendMessageInput) (*Message, *apierror.APIError)
	// ListMessages returns a conversation's messages, keyset-paginated by sequence.
	ListMessages(ctx context.Context, input ListMessagesInput) (*MessagePage, *apierror.APIError)
	// MarkConversationRead advances the caller's read cursor and returns the resulting unread count.
	MarkConversationRead(ctx context.Context, conversationID string, upToSequence int64) (int64, *apierror.APIError)
	// IsParticipant reports whether the user (resolved to their account_user within accountID)
	// is an active participant of the conversation (WS subscribe authz).
	IsParticipant(ctx context.Context, conversationID, userID, accountID string) (bool, *apierror.APIError)
	// SendTyping broadcasts an ephemeral typing indicator from the caller to the conversation's live subscribers (the conv: topic). Best-effort and not persisted: it is published directly to the realtime fanout, bypassing the transactional outbox. The caller must be an active participant.
	SendTyping(ctx context.Context, conversationID string) *apierror.APIError

	// ── Phase 3: group management, lifecycle, edit/delete, blocks ──

	// UpdateConversation renames a group and/or sets its lifecycle status (owner/admin).
	UpdateConversation(ctx context.Context, conversationID string, title *string, status *string, clearTitle bool) (*Conversation, *apierror.APIError)
	// AddParticipant adds (or reactivates) a member in a group (owner/admin).
	AddParticipant(ctx context.Context, conversationID, accountUserID, role string) (*Conversation, *apierror.APIError)
	// RemoveParticipant removes a member from a group by participant id (owner/admin).
	RemoveParticipant(ctx context.Context, conversationID, participantID string) *apierror.APIError
	// UpdateParticipantRole changes a member's role (owner only).
	UpdateParticipantRole(ctx context.Context, conversationID, participantID, role string) (*Conversation, *apierror.APIError)
	// Leave removes the caller from a conversation (state=left, hidden).
	Leave(ctx context.Context, conversationID string) *apierror.APIError
	// Hide soft-hides the conversation for the caller.
	Hide(ctx context.Context, conversationID string) *apierror.APIError
	// Unhide clears the caller's hidden flag on an active conversation.
	Unhide(ctx context.Context, conversationID string) *apierror.APIError
	// SetMute toggles the caller's per-conversation mute.
	SetMute(ctx context.Context, conversationID string, muted bool, mutedUntil *time.Time) (*Conversation, *apierror.APIError)
	// Block prevents the target from DMing the caller (and vice-versa).
	Block(ctx context.Context, blockedAccountUserID string) (*MessageBlock, *apierror.APIError)
	// Unblock removes a block the caller created.
	Unblock(ctx context.Context, blockedAccountUserID string) *apierror.APIError
	// ListBlocks lists the account_users the caller has blocked.
	ListBlocks(ctx context.Context) ([]*MessageBlock, *apierror.APIError)

	// ── Messaging directory + abuse reports (§12.4, §12.10) ──

	// ListContacts returns the caller's messageable targets (the messaging directory). Internal actors get active account_users in their account filtered by the name substring query (including themselves); customer relation actors get a single "support" contact.
	ListContacts(ctx context.Context, query string) ([]*MessagingContact, *apierror.APIError)
	// ReportConversation persists a minimal abuse report against a conversation (optionally a specific message). The caller must be an active participant of the conversation.
	ReportConversation(ctx context.Context, conversationID string, messageID *string, reason string) (*MessageReport, *apierror.APIError)

	// ── Phase 4: attachment upload pipeline ──

	// CreateAttachmentUploadURL issues a presigned PUT target for a chat attachment. The caller must be an active participant of the conversation. The returned s3_key is echoed back in the send-message request once the client has uploaded the object.
	CreateAttachmentUploadURL(ctx context.Context, conversationID, filename, contentType string) (*AttachmentUploadTarget, *apierror.APIError)

	// ── Phase 5: scheduled messages ──

	// ScheduleMessage queues a message for delivery at a future time (active participant, post role). It
	// is a message row at status=scheduled, promoted to a sent timeline message in place when due.
	ScheduleMessage(ctx context.Context, input CreateScheduledMessageInput) (*Message, *apierror.APIError)
	// ListScheduledMessages returns the caller's scheduled (not-yet-sent) messages in a conversation.
	ListScheduledMessages(ctx context.Context, conversationID string) ([]*Message, *apierror.APIError)
	// CancelScheduledMessage cancels a scheduled message the caller created.
	CancelScheduledMessage(ctx context.Context, id string) (*Message, *apierror.APIError)
	// DeliverDueScheduledMessages materializes all currently-due scheduled messages into real messages. Called by the lease-guarded scheduler worker (no request identity in context).
	DeliverDueScheduledMessages(ctx context.Context, limit int32) (int, *apierror.APIError)

	// ── Phase 5: agents as participants ──

	// AddAgentParticipant adds (or re-activates) an agent participant in a group conversation with a trigger policy (owner/admin).
	AddAgentParticipant(ctx context.Context, input AddAgentParticipantInput) (*ConversationParticipant, *apierror.APIError)
	// RemoveAgentParticipant removes an agent participant from a conversation by participant id (owner/admin).
	RemoveAgentParticipant(ctx context.Context, conversationID, participantID string) *apierror.APIError

	// PostAgentReply posts an agent run's reply into a conversation as the agent participant (kind=agent, sender_participant_id resolved from the agent_config, message.agent_run_id linked). Service-internal (no request identity); idempotent on the client message id. The legacy single-shot path (create and complete in one call); the streaming path uses Start/Patch/Complete below.
	PostAgentReply(ctx context.Context, input AgentReplyInput) *apierror.APIError

	// PostAgentReplyStart posts a streaming agent reply: an empty kind=agent message in streaming_state, so the thread renders the bubble immediately (it fills in via patches). No bell — deferred to Complete. Idempotent on the (agent-owned) message id. Called by the agent-reply consumer (phase=start).
	PostAgentReplyStart(ctx context.Context, input AgentReplyInput) *apierror.APIError

	// PostAgentReplyComplete finalizes a streaming reply: sets the completed body, flips streaming_state to complete, pushes the final body, and fires the bell. Falls back to creating the message outright when the start was lost. Idempotent. Called by the agent-reply consumer (phase=final / legacy empty).
	PostAgentReplyComplete(ctx context.Context, input AgentReplyInput) *apierror.APIError

	// PostAgentReplyPatch streams a partial body into an in-flight reply (server-push-only message.updated, no bell, best-effort). Called by the agent-reply-patch consumer.
	PostAgentReplyPatch(ctx context.Context, input AgentReplyPatchInput) *apierror.APIError

	// PostConversationSystemEvent appends a senderless system_event message (the body carries the whole sentence, e.g. "Dane approved update_customer") that renders as a timeline divider. Service-internal, timeline-only (no bell), idempotent on the client message id. Used for tool-approval notices.
	PostConversationSystemEvent(ctx context.Context, input SystemEventInput) *apierror.APIError

	// IngestInboundEmail threads a parsed inbound email into the conversation bound to its inbox (creating the conversation on the first message of a thread), records it in the email_message ledger for dedup, and dispatches to the inbox's agent. Idempotent on the rfc Message-ID.
	IngestInboundEmail(ctx context.Context, input IngestInboundEmailInput) *apierror.APIError

	// SendInboxReply sends an agent's outbound email through the conversation's bound inbox (SES, threaded under the latest inbound mail), records it in the email_message ledger, and posts it as a message in the thread. The mutating, externally-visible half of the support workflow.
	SendInboxReply(ctx context.Context, input SendInboxReplyInput) (*Message, *apierror.APIError)
	// PostReplyDraft proposes an agent's customer reply as a real status=draft message held for human
	// approval (channel resolved from the case). Surfaces in the reply-drafts bar. Called by agent-service.
	PostReplyDraft(ctx context.Context, input PostReplyDraftInput) (*Message, *apierror.APIError)

	// ── External customer-service cases: reply, triage, assignment, links, drafts ──

	// UpdateWorkflowStatus sets an external case's triage lane (new|open|waiting_*|needs_approval|resolved).
	UpdateWorkflowStatus(ctx context.Context, conversationID, status string) (*Conversation, *apierror.APIError)
	// AssignConversation sets (or clears, with nils) the single polymorphic owner (user or team) for an external case.
	AssignConversation(ctx context.Context, conversationID string, assigneeResourceType, assigneeResourceID *string) (*Conversation, *apierror.APIError)
	// ListInbox lists external (customer-facing) cases for the support inbox, filtered for triage.
	ListInbox(ctx context.Context, input SupportInboxInput) (*ConversationPage, *apierror.APIError)
	// AddConversationLink links a business record to a conversation (idempotent on duplicate).
	AddConversationLink(ctx context.Context, conversationID, resourceType, resourceID string) (*ConversationLink, *apierror.APIError)
	// RemoveConversationLink removes a business-record link from a conversation by link id.
	RemoveConversationLink(ctx context.Context, conversationID, linkID string) *apierror.APIError
	// ListConversationLinks lists a conversation's secondary business-record links.
	ListConversationLinks(ctx context.Context, conversationID string) ([]*ConversationLink, *apierror.APIError)
	// ListConversationsByResource lists conversations linked to a business record (topic anchor or link).
	ListConversationsByResource(ctx context.Context, resourceType, resourceID string, limit int32) ([]*Conversation, *apierror.APIError)
	// CreateReplyDraft proposes a customer-reply draft on an external case (draft-first; not sent). The
	// draft is a message row at status=draft.
	CreateReplyDraft(ctx context.Context, input CreateReplyDraftInput) (*Message, *apierror.APIError)
	// ListReplyDrafts lists a case's reply drafts (optionally filtered by status).
	ListReplyDrafts(ctx context.Context, conversationID string, status *string) ([]*Message, *apierror.APIError)
	// UpdateReplyDraft edits a still-open draft's body/subject.
	UpdateReplyDraft(ctx context.Context, draftID, body string, subject *string) (*Message, *apierror.APIError)
	// RejectReplyDraft discards an open draft without sending.
	RejectReplyDraft(ctx context.Context, draftID string) (*Message, *apierror.APIError)
	// ApproveAndSendReplyDraft promotes an open draft to the customer (portal or email) in place: it
	// becomes exactly one customer-visible sent message, attributed to the case's alias persona.
	ApproveAndSendReplyDraft(ctx context.Context, draftID, clientMessageID string) (*Message, *apierror.APIError)

	// ── Phase 6: GDPR redaction + legal hold ──

	// SetLegalHold sets or clears a conversation's legal-hold flag, exempting it from the reaper and from redaction while held. Internal actor with messaging update permission only.
	SetLegalHold(ctx context.Context, conversationID string, hold bool) (*Conversation, *apierror.APIError)
	// RedactConversation strips the body/preview of every message and deletes all attachments while keeping the message rows as an audit shell. Refuses while the conversation is under legal hold.
	// Internal actor with messaging delete permission only.
	RedactConversation(ctx context.Context, conversationID string) (*Conversation, *apierror.APIError)

	// ── Reusable groups (rosters): named member sets that seed many conversations ──

	// CreateMessagingGroup creates a named roster of users and/or agents in the caller's account.
	CreateMessagingGroup(ctx context.Context, input CreateMessagingGroupInput) (*MessagingGroup, *apierror.APIError)
	// ListMessagingGroups returns the account's rosters (with members), most-recently-updated first.
	ListMessagingGroups(ctx context.Context) ([]*MessagingGroup, *apierror.APIError)
	// GetMessagingGroup returns one roster (with members) in the caller's account.
	GetMessagingGroup(ctx context.Context, groupID string) (*MessagingGroup, *apierror.APIError)
	// UpdateMessagingGroup renames a roster.
	UpdateMessagingGroup(ctx context.Context, groupID, name string) (*MessagingGroup, *apierror.APIError)
	// DeleteMessagingGroup deletes a roster, nulling group_id on any conversations it seeded.
	DeleteMessagingGroup(ctx context.Context, groupID string) *apierror.APIError
	// AddMessagingGroupMember adds a user or agent to a roster and returns the updated roster.
	AddMessagingGroupMember(ctx context.Context, input AddMessagingGroupMemberInput) (*MessagingGroup, *apierror.APIError)
	// RemoveMessagingGroupMember removes a member (by member id) from a roster and returns the updated roster.
	RemoveMessagingGroupMember(ctx context.Context, groupID, memberID string) (*MessagingGroup, *apierror.APIError)
}

// ListConversationsInput parameterizes the caller's conversation list (cursor opaque).
type ListConversationsInput struct {
	Cursor *string
	Limit  int32
	Type   *string
	Status string
}

// ListMessagesInput parameterizes a conversation's message history (cursor opaque; the cursor encodes a before_sequence for loading older history).
type ListMessagesInput struct {
	ConversationID string
	Cursor         *string
	Limit          int32
	AfterSequence  *int64
}

type NotificationSvc interface {
	// SendEmail sends an email to the specified recipients.
	//
	// Returns the provider message ID if the email is successfully accepted for delivery.
	SendEmail(ctx context.Context, data EmailSendData) (*string, *apierror.APIError)

	// LogEmail records an email in persistent storage.
	//
	// Behavior:
	//   - If the email has already been logged, the operation is a no-op.
	LogEmail(ctx context.Context, data EmailLogData) *apierror.APIError

	// LogFailedEmail records an email that could not be sent, keyed on the outbox message ID so delivery retries do not each append a row.
	//
	// Behavior:
	//   - If the email has already been logged for this message ID, the operation is a no-op.
	LogFailedEmail(ctx context.Context, messageID string, data EmailSendData) *apierror.APIError

	// SendEnterpriseRequest sends an enterprise upgrade request email to support.
	SendEnterpriseRequest(ctx context.Context, req *EnterpriseRequestData) *apierror.APIError
}

// MessagingSvc owns the in-app notification (bell) feature. Read/mark operations are scoped to the calling actor (recipient = identity.Actor.ID). SendNotification enqueues a fan-out intent; FanOut is the consumer-side handler that materializes notification rows and emits the realtime push.
type MessagingSvc interface {
	// SendNotification enqueues a notification fan-out (targeted at a user). Admin/internal only.
	SendNotification(ctx context.Context, input SendNotificationInput) (int64, *apierror.APIError)
	// ListNotifications returns the caller's feed, cursor-paginated.
	ListNotifications(ctx context.Context, input ListNotificationsInput) (*NotificationPage, *apierror.APIError)
	// GetNotification returns one notification owned by the caller.
	GetNotification(ctx context.Context, id string) (*Notification, *apierror.APIError)
	// GetUnreadCount returns the caller's unread tallies.
	GetUnreadCount(ctx context.Context) (*UnreadCounts, *apierror.APIError)
	// GetUnreadSummary returns the caller's unread totals across every account they belong to.
	GetUnreadSummary(ctx context.Context) (*UnreadSummary, *apierror.APIError)
	// MarkSeen/Read/Dismissed transition the caller's notification (idempotent).
	MarkSeen(ctx context.Context, id string) (*Notification, *apierror.APIError)
	MarkRead(ctx context.Context, id string) (*Notification, *apierror.APIError)
	MarkDismissed(ctx context.Context, id string) (*Notification, *apierror.APIError)
	// MarkAllSeen marks every unseen notification for the caller as seen.
	MarkAllSeen(ctx context.Context) (int64, *apierror.APIError)
	// ListAnnouncements returns the caller's active announcements, cursor-paginated.
	ListAnnouncements(ctx context.Context, input ListAnnouncementsInput) (*AnnouncementPage, *apierror.APIError)
	// GetAnnouncement returns one active announcement visible to the caller.
	GetAnnouncement(ctx context.Context, id string) (*Announcement, *apierror.APIError)
	// MarkAnnouncementSeen/Read/Dismissed transition the caller's receipt (idempotent).
	MarkAnnouncementSeen(ctx context.Context, id string) (*Announcement, *apierror.APIError)
	MarkAnnouncementRead(ctx context.Context, id string) (*Announcement, *apierror.APIError)
	MarkAnnouncementDismissed(ctx context.Context, id string) (*Announcement, *apierror.APIError)
	// FanOut materializes notification rows for an alert/message intent and emits the realtime push. Called by the fan-out consumer. dedupeSeed (the source message id) makes per-recipient inserts idempotent across redelivery.
	FanOut(ctx context.Context, dedupeSeed string, data messaging.AlertFanoutData) *apierror.APIError
	// NotifyCustomerRegistered fans a bell notification out to the seller's customer-service support-route group when a buyer registers on the portal. Called by the customer-registered consumer. seed makes per-recipient inserts idempotent across redelivery.
	NotifyCustomerRegistered(ctx context.Context, seed string, data messaging.CustomerRegisteredData) *apierror.APIError
	// ListNotificationPreferences returns the caller's channel preferences (global + per-category).
	ListNotificationPreferences(ctx context.Context) ([]*NotificationPreference, *apierror.APIError)
	// UpsertNotificationPreference creates or replaces the caller's preference for a category.
	UpsertNotificationPreference(ctx context.Context, input UpsertNotificationPreferenceInput) (*NotificationPreference, *apierror.APIError)
}

// SendNotificationInput is the validated input for an admin/system notification send. The target is polymorphic: TargetType ("account_user" | "account") + TargetID determine whether the send is a per-user notification or an account-wide broadcast announcement.
type SendNotificationInput struct {
	Category         string
	TargetType       string
	TargetID         string
	Title            string
	Body             *string
	TemplateKey      *string
	TemplateParams   json.RawMessage
	LinkResourceType *string
	LinkResourceID   *string
	Priority         *string
}

// ListNotificationsInput parameterizes the caller's feed query (cursor opaque to callers).
type ListNotificationsInput struct {
	Cursor   *string
	Limit    int32
	Category *string
	// Status filters by lifecycle state ("unseen"/"seen"/"read"/"dismissed"); nil = the default active feed (all non-dismissed).
	Status *string
	// Search is a free-text term matched against title/body. nil = no search.
	Search *string
	// SenderIDs / SenderTypes filter by sender attribution (multi-value). Empty = no filter.
	SenderIDs   []string
	SenderTypes []string
}

// NotificationPage is one page of the caller's feed plus the next opaque cursor.
type NotificationPage struct {
	Notifications []*Notification
	NextCursor    *string
	HasNextPage   bool
}

// ListAnnouncementsInput parameterizes the caller's announcement feed (cursor opaque).
type ListAnnouncementsInput struct {
	Cursor *string
	Limit  int32
}

type EmailLogData struct {
	SesMessageID string   `json:"ses_message_id"`
	To           []string `json:"to,omitempty"`
	AccountID    *string  `json:"account_id,omitempty"`
	SentByID     *string  `json:"sent_by_id,omitempty"`
	Subject      string   `json:"subject"`
	Filename     *string  `json:"filename,omitempty"`
}

type EmailSendData struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	SendAs  *string  `json:"send_as,omitempty"`
	// TemplateID is carried past rendering because it decides whose identity the message goes out under: only a merchant's own correspondence may send from that merchant's domain.
	TemplateID constants.EmailTemplate `json:"template_id,omitempty"`
	AccountID  *string                 `json:"account_id,omitempty"`
	SentByID   *string                 `json:"sent_by_id,omitempty"`
	// Attachment fields for raw MIME emails.
	Attachment []byte  `json:"attachment,omitempty"`
	Filename   *string `json:"filename,omitempty"`
}

// EnterpriseRequestData contains data for an enterprise upgrade request email
type EnterpriseRequestData struct {
	AccountID       string
	AccountName     string
	CurrentPlanName string
	RequesterName   string
	RequesterEmail  string
}

type EmailSender interface {
	Send(ctx context.Context, data EmailData) (*string, *apierror.APIError)
}

type EmailData struct {
	To         []string `json:"to"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body"`
	SendAs     *string  `json:"send_as,omitempty"`
	Attachment []byte   `json:"attachment,omitempty"`
	Filename   *string  `json:"filename,omitempty"`
	// Email-bridge fields. From overrides the default noreply@ sender (e.g. a customer inbox address); Cc/InReplyTo/References/MessageID carry rfc822 threading; PlainText sends text/plain not html.
	From       *string  `json:"from,omitempty"`
	Cc         []string `json:"cc,omitempty"`
	InReplyTo  *string  `json:"in_reply_to,omitempty"`
	References *string  `json:"references,omitempty"`
	MessageID  *string  `json:"message_id,omitempty"`
	PlainText  bool     `json:"plain_text,omitempty"`
}
