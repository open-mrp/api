package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type EmailLogRepo interface {
	Create(ctx context.Context, emailLog *EmailLog) *apierror.APIError
	FindBySesMessageID(ctx context.Context, sesMessageID string) (*EmailLog, *apierror.APIError)
}

// EmailDomainRepo persists customer-owned email domains and their SES verification state.
type EmailDomainRepo interface {
	Create(ctx context.Context, id, accountID string, input *CreateEmailDomainInput) *apierror.APIError
	GetByID(ctx context.Context, id, accountID string) (*EmailDomain, *apierror.APIError)
	GetByDomain(ctx context.Context, domain string) (*EmailDomain, *apierror.APIError)
	ListByAccount(ctx context.Context, accountID string) ([]*EmailDomain, *apierror.APIError)
	MarkVerified(ctx context.Context, id, accountID string) *apierror.APIError
	UpdateStatus(ctx context.Context, id, accountID, status string, dkimTokens []string) *apierror.APIError
	// Delete removes a domain; deleted is false when no matching domain existed.
	Delete(ctx context.Context, id, accountID string) (deleted bool, apiErr *apierror.APIError)
}

// EmailInboxRepo persists routable inbox addresses and resolves the inbox an inbound mail arrived at.
type EmailInboxRepo interface {
	Create(ctx context.Context, id, accountID string, input *CreateEmailInboxInput) *apierror.APIError
	GetByID(ctx context.Context, id, accountID string) (*EmailInbox, *apierror.APIError)
	// GetByAddress resolves the inbox for inbound routing; address is globally unique (no account scope).
	GetByAddress(ctx context.Context, address string) (*EmailInbox, *apierror.APIError)
	// GetByIDSystem resolves an inbox by id for inbound routing (no account scope), used to match mail
	// delivered to a per-inbox forwarding address (<inbox_id>@<inbound domain>). IDs are globally unique.
	GetByIDSystem(ctx context.Context, id string) (*EmailInbox, *apierror.APIError)
	ListByAccount(ctx context.Context, accountID string) ([]*EmailInbox, *apierror.APIError)
	// CountByDomain returns how many inboxes are bound to a domain, so a domain delete can be blocked while inboxes exist.
	CountByDomain(ctx context.Context, accountID, emailDomainID string) (int64, *apierror.APIError)
	Update(ctx context.Context, id, accountID string, input *UpdateEmailInboxInput) *apierror.APIError
	Delete(ctx context.Context, id, accountID string) (bool, *apierror.APIError)
}

// EmailMessageRepo is the per-rfc822 ledger backing threading + at-least-once dedup.
type EmailMessageRepo interface {
	// TryInsert returns inserted=false when the rfc_message_id already exists (redelivery), so the inbound consumer can ack and skip without re-threading.
	TryInsert(ctx context.Context, input *CreateEmailMessageInput) (inserted bool, apiErr *apierror.APIError)
	GetByRfcID(ctx context.Context, rfcMessageID string) (*EmailMessage, *apierror.APIError)
	// FindThreadConversation resolves the conversation an inbound mail belongs to from its In-Reply-To
	// + References message-ids. Returns ErrorCodeResourceNotFound when no prior email matches.
	FindThreadConversation(ctx context.Context, rfcMessageIDs []string) (*EmailThreadMatch, *apierror.APIError)
	GetLatestInbound(ctx context.Context, conversationID string) (*EmailMessage, *apierror.APIError)
}

// SupportRouteRepo persists the group conversation designated to handle a relationship's support and resolves it for an inbound customer (per-relation override winning over the account default).
type SupportRouteRepo interface {
	Upsert(ctx context.Context, id, accountID, relationAccountID, groupConversationID string) *apierror.APIError
	// Get returns the route for an exact scope (relationAccountID "" = account default), or nil when none.
	Get(ctx context.Context, accountID, relationAccountID string) (*SupportRoute, *apierror.APIError)
	// Resolve returns the effective route for a customer: the per-relation override if present, else the account default. Returns nil when neither is configured.
	Resolve(ctx context.Context, accountID, relationAccountID string) (*SupportRoute, *apierror.APIError)
	Delete(ctx context.Context, accountID, relationAccountID string) (bool, *apierror.APIError)
}

// NotificationRepo persists and queries per-user bell-feed notifications. Mark* methods are idempotent (set-if-null) and return the updated row; ownership is enforced by requiring the recipient on every read/mutate.
type NotificationRepo interface {
	Create(ctx context.Context, notification *Notification) *apierror.APIError
	// CreateBatch inserts many notifications (fan-out). Call within a transaction.
	CreateBatch(ctx context.Context, notifications []*Notification) *apierror.APIError
	GetByID(ctx context.Context, id, recipientAccountUserID string) (*Notification, *apierror.APIError)
	List(ctx context.Context, filter NotificationListFilter) ([]*Notification, *apierror.APIError)
	CountUnseen(ctx context.Context, recipientAccountUserID string) (int64, *apierror.APIError)
	MarkSeen(ctx context.Context, id, recipientAccountUserID string) (*Notification, *apierror.APIError)
	MarkRead(ctx context.Context, id, recipientAccountUserID string) (*Notification, *apierror.APIError)
	MarkDismissed(ctx context.Context, id, recipientAccountUserID string) (*Notification, *apierror.APIError)
	MarkAllSeen(ctx context.Context, recipientAccountUserID string) (int64, *apierror.APIError)
	// DismissBySourceMessage withdraws every bell notification that projected a deleted message (§12.7).
	DismissBySourceMessage(ctx context.Context, accountID, sourceMessageID string) *apierror.APIError
	// DismissByConversation withdraws a recipient's bell notifications for a conversation once they read it in-thread, so already-seen chat alerts don't stack. Returns the number dismissed.
	DismissByConversation(ctx context.Context, recipientAccountUserID, conversationID string) (int64, *apierror.APIError)
	// ResolveAccountUserID maps (user_id, account_id) to the account_user id used as the notification recipient key. identity.Actor.ID is the user id, not the account_user id.
	ResolveAccountUserID(ctx context.Context, userID, accountID string) (string, *apierror.APIError)
	// ResolveUserID maps an account_user id back to its user id (us_). The realtime push targets the WS user-topic, which the gateway keys by user id, not account_user id.
	ResolveUserID(ctx context.Context, accountUserID string) (string, *apierror.APIError)
	// CountUnseenByUserAccounts aggregates the user's unseen notifications per account across every account they belong to (cross-account unread hint).
	CountUnseenByUserAccounts(ctx context.Context, userID string) ([]AccountUnread, *apierror.APIError)
	// ResolveRecipientContact returns a recipient's email + display name for the email bridge.
	ResolveRecipientContact(ctx context.Context, accountUserID string) (*RecipientContact, *apierror.APIError)
	// ListMessagingContacts returns active account_users in an account whose name matches the
	// (case-insensitive) substring query, for the messaging directory. An empty query matches all.
	ListMessagingContacts(ctx context.Context, accountID, query string) ([]*MessagingContact, *apierror.APIError)
}

// RecipientContact carries a recipient's email + display name for the email bridge.
type RecipientContact struct {
	Email string
	Name  string
}

// AnnouncementRepo persists and queries broadcast announcements and the caller's per-user receipt state. Mark* methods lazily upsert a receipt and are idempotent (set-if-null).
type AnnouncementRepo interface {
	Create(ctx context.Context, id string, input *CreateAnnouncementInput) *apierror.APIError
	GetActiveByID(ctx context.Context, id, accountUserID string, accountID *string) (*Announcement, *apierror.APIError)
	ListActive(ctx context.Context, filter AnnouncementListFilter) ([]*Announcement, *apierror.APIError)
	CountUnseen(ctx context.Context, accountUserID string, accountID *string) (int64, *apierror.APIError)
	MarkSeen(ctx context.Context, announcementID, accountUserID string) *apierror.APIError
	MarkRead(ctx context.Context, announcementID, accountUserID string) *apierror.APIError
	MarkDismissed(ctx context.Context, announcementID, accountUserID string) *apierror.APIError
	// CountUnseenByUserAccounts aggregates the user's unseen account-scoped announcements per account across every account they belong to (cross-account unread hint).
	CountUnseenByUserAccounts(ctx context.Context, userID string) ([]AccountUnread, *apierror.APIError)
}

// ConversationRepo persists and queries conversations, their DM-dedup keys, and the caller's membership scoping. Sequence allocation (LockSequence + AdvanceAfterMessage) must run inside a send transaction.
type ConversationRepo interface {
	Create(ctx context.Context, id string, input *CreateConversationInput, accountID string) *apierror.APIError
	GetByID(ctx context.Context, id, accountID string) (*Conversation, *apierror.APIError)
	ListForUser(ctx context.Context, filter ConversationListFilter) ([]*Conversation, *apierror.APIError)
	// LockSequence locks the conversation row and returns the next sequence to assign.
	LockSequence(ctx context.Context, id, accountID string) (int64, *apierror.APIError)
	// AdvanceAfterMessage bumps next_sequence and updates the last-message denormalization.
	AdvanceAfterMessage(ctx context.Context, id, lastMessageID string, lastMessageAt time.Time) *apierror.APIError
	// BindInbox marks a conversation as bridged to an email inbox (sets email_inbox_id + external address).
	BindInbox(ctx context.Context, id, accountID, inboxID, externalAddress string) *apierror.APIError
	// GetDMConversationID returns the existing conversation id for a DM key, or not-found.
	GetDMConversationID(ctx context.Context, accountID, dmKey string) (string, *apierror.APIError)
	// CreateDMKey reserves the DM key; a duplicate-entry error means the DM already exists.
	CreateDMKey(ctx context.Context, accountID, dmKey, conversationID string) error
	// Update applies a partial update (title/archive) to a conversation.
	Update(ctx context.Context, id, accountID string, title *string, isArchived *bool, clearTitle bool) *apierror.APIError
	// GetCustomerSupport returns the vendor account's portal support case for a customer account, or not-found.
	GetCustomerSupport(ctx context.Context, vendorAccountID, customerAccountID string) (*Conversation, *apierror.APIError)
	// CreateCustomerSupport creates a customer-audience group case owned by the vendor account (portal contact support), anchoring its topic to the customer account so the case surfaces the customer's name.
	CreateCustomerSupport(ctx context.Context, id, vendorAccountID, customerAccountID string) *apierror.APIError
	// SetLegalHold sets or clears the legal-hold flag (exempts the conversation from reaper/redaction).
	SetLegalHold(ctx context.Context, id, accountID string, hold bool) *apierror.APIError
	// RedactMessages clears the body/preview of every live message, keeping the rows as audit shells.
	// Returns the number of messages redacted.
	RedactMessages(ctx context.Context, conversationID string) (int64, *apierror.APIError)
	// PromoteToCustomerCase flips a conversation to audience=customer (seeding workflow_status=new), used when an inbound email opens a new external thread.
	PromoteToCustomerCase(ctx context.Context, id, accountID string) *apierror.APIError
	// SetWorkflowStatus sets the customer-service triage lane.
	SetWorkflowStatus(ctx context.Context, id, accountID, status string) *apierror.APIError
	// Assign sets (or clears, with nils) the owning user and/or team for a case.
	Assign(ctx context.Context, id, accountID string, assigneeAccountUserID, assignedGroupID *string) *apierror.APIError
	// ListInbox lists external (audience=customer) cases for the support inbox.
	ListInbox(ctx context.Context, filter SupportInboxFilter) ([]*Conversation, *apierror.APIError)
	// ListByResource lists conversations linked to a business record (via topic anchor or a link row).
	ListByResource(ctx context.Context, accountID, resourceType, resourceID string, limit int32) ([]*Conversation, *apierror.APIError)
}

// ConversationLinkRepo persists secondary business-record links on conversations.
type ConversationLinkRepo interface {
	// Create adds a link; a duplicate (conversation, type, id) is mapped to a conflict error.
	Create(ctx context.Context, l *ConversationLink) *apierror.APIError
	// Delete removes a link by id (scoped to its conversation and account); removed is false if no matching link existed.
	Delete(ctx context.Context, linkID, conversationID, accountID string) (removed bool, apiErr *apierror.APIError)
	List(ctx context.Context, conversationID, accountID string) ([]*ConversationLink, *apierror.APIError)
}

// MessagingGroupRepo persists reusable rosters (messaging_group) and their members.
type MessagingGroupRepo interface {
	// Create inserts a roster row (members are inserted separately via AddMember).
	Create(ctx context.Context, g *MessagingGroup) *apierror.APIError
	// Get returns a roster (without members), or not-found.
	Get(ctx context.Context, id, accountID string) (*MessagingGroup, *apierror.APIError)
	// List returns the account's rosters (without members), most-recently-updated first.
	List(ctx context.Context, accountID string) ([]*MessagingGroup, *apierror.APIError)
	// Rename updates a roster's name; renamed is false when no matching roster existed.
	Rename(ctx context.Context, id, accountID, name string) (renamed bool, apiErr *apierror.APIError)
	// Touch bumps updated_at so the roster re-sorts after a membership change.
	Touch(ctx context.Context, id, accountID string) *apierror.APIError
	// Delete removes a roster; deleted is false when no matching roster existed.
	Delete(ctx context.Context, id, accountID string) (deleted bool, apiErr *apierror.APIError)
	// AddMember inserts a member; a duplicate (group, user/agent) is mapped to a conflict error.
	AddMember(ctx context.Context, m *MessagingGroupMember) *apierror.APIError
	// ListMembers returns a roster's members in insertion order.
	ListMembers(ctx context.Context, groupID string) ([]*MessagingGroupMember, *apierror.APIError)
	// RemoveMember deletes a member by id; removed is false when no matching member existed.
	RemoveMember(ctx context.Context, memberID, groupID string) (removed bool, apiErr *apierror.APIError)
	// DeleteMembers removes every member of a roster (used on group delete; no FK cascade under relationMode=prisma).
	DeleteMembers(ctx context.Context, groupID string) *apierror.APIError
	// ClearConversationGroup nulls group_id on every conversation that was seeded from a roster (on delete).
	ClearConversationGroup(ctx context.Context, accountID, groupID string) *apierror.APIError
}

// ParticipantRepo persists conversation membership and per-user read cursors.
type ParticipantRepo interface {
	Create(ctx context.Context, p *ConversationParticipant) *apierror.APIError
	// Get returns the caller's participant row, or not-found if they are not a member.
	Get(ctx context.Context, conversationID, accountUserID string) (*ConversationParticipant, *apierror.APIError)
	List(ctx context.Context, conversationID string) ([]*ConversationParticipant, *apierror.APIError)
	// ListAll returns every participant regardless of state, for resolving historical message authorship (a left/removed member still authored their past messages).
	ListAll(ctx context.Context, conversationID string) ([]*ConversationParticipant, *apierror.APIError)
	// AdvanceReadCursor moves the caller's read cursor forward (never rewinds).
	AdvanceReadCursor(ctx context.Context, conversationID, accountUserID, lastReadMessageID string, sequence int64) *apierror.APIError
	// GetByID returns a participant by its id within a conversation.
	GetByID(ctx context.Context, participantID, conversationID string) (*ConversationParticipant, *apierror.APIError)
	// CreateAgent adds an agent participant (participant_type=agent) with a trigger policy.
	CreateAgent(ctx context.Context, id, accountID string, input *AddAgentParticipantInput) *apierror.APIError
	// GetByAgentConfigID returns an agent participant by its agent_config_id, or not-found.
	GetByAgentConfigID(ctx context.Context, conversationID, agentConfigID string) (*ConversationParticipant, *apierror.APIError)
	// ReactivateAgent re-activates a previously-removed agent participant and updates its trigger policy.
	ReactivateAgent(ctx context.Context, participantID string, input *AddAgentParticipantInput) *apierror.APIError
	// SetStateByID changes a participant's state by participant id (used for agent participants, which have no account_user_id key).
	SetStateByID(ctx context.Context, participantID, conversationID, state string) *apierror.APIError
	// CreateCustomer adds a customer relation participant: keyed by the customer's account (routing/dedup) and carrying the customer-account user who opened the case (surfaced as the participant's user actor).
	CreateCustomer(ctx context.Context, id, conversationID, vendorAccountID, customerAccountID, accountUserID string) *apierror.APIError
	// GetByRelationAccount returns the customer participant for a customer account, or not-found.
	GetByRelationAccount(ctx context.Context, conversationID, customerAccountID string) (*ConversationParticipant, *apierror.APIError)
	// AdvanceReadCursorByID advances a participant's read cursor by participant id (forward-only).
	AdvanceReadCursorByID(ctx context.Context, participantID, lastReadMessageID string, sequence int64) *apierror.APIError
	// SetRole changes a participant's role.
	SetRole(ctx context.Context, conversationID, accountUserID, role string) *apierror.APIError
	// SetState changes a participant's membership (e.g. removed).
	SetState(ctx context.Context, conversationID, accountUserID, membership string) *apierror.APIError
	// Leave marks the participant as left and hidden.
	Leave(ctx context.Context, conversationID, accountUserID string) *apierror.APIError
	// Hide soft-hides the conversation for the participant (membership unchanged).
	Hide(ctx context.Context, conversationID, accountUserID string) *apierror.APIError
	// Unhide clears the participant's hidden flag (only when membership=active).
	Unhide(ctx context.Context, conversationID, accountUserID string) *apierror.APIError
	// Reactivate re-activates a previously left/removed participant with the given role.
	Reactivate(ctx context.Context, conversationID, accountUserID, role string) *apierror.APIError
	// SetMute toggles per-user mute (with an optional expiry).
	SetMute(ctx context.Context, conversationID, accountUserID string, isMuted bool, mutedUntil *time.Time) *apierror.APIError
}

// MessageRepo persists and queries chat messages.
type MessageRepo interface {
	// Create inserts a message. inserted is false when the client dedupe key already exists
	// (a concurrent idempotent resend lost the race); the caller resolves the winner.
	Create(ctx context.Context, m *Message) (inserted bool, apiErr *apierror.APIError)
	GetByID(ctx context.Context, id, accountID string) (*Message, *apierror.APIError)
	// GetByIDs batch-loads messages by id (for last-message previews on the conversation list).
	GetByIDs(ctx context.Context, ids []string) ([]*Message, *apierror.APIError)
	// GetByClientID resolves an existing message by its client dedupe key (idempotent send).
	GetByClientID(ctx context.Context, conversationID, clientMessageID string) (*Message, *apierror.APIError)
	List(ctx context.Context, filter MessageListFilter) ([]*Message, *apierror.APIError)
	// GetLastVisible returns the most recent customer-visible (non-internal) message in a conversation, for a customer viewer's last-message preview; nil when none exist.
	GetLastVisible(ctx context.Context, conversationID string) (*Message, *apierror.APIError)
	// CountVisibleAfter counts customer-visible messages after a read cursor (a customer's unread badge).
	CountVisibleAfter(ctx context.Context, conversationID string, afterSequence int64) (int64, *apierror.APIError)
	// UpdateBody edits a message's body/preview and stamps edited_at.
	UpdateBody(ctx context.Context, id, accountID string, body, preview *string) *apierror.APIError
	// SetStreamingBody streams a partial/final body into an in-flight agent reply (without marking it edited), optionally flipping streaming_state to "complete". applied is false when the row is no longer streaming (already complete/deleted) or doesn't exist — the caller skips the realtime push.
	SetStreamingBody(ctx context.Context, id, accountID string, body, preview *string, state string) (applied bool, apiErr *apierror.APIError)
	// SetMessageMetadata overwrites a message's metadata JSON — used to annotate a finalized agent reply with a failure marker + error code so the client can flag it and react.
	SetMessageMetadata(ctx context.Context, id, accountID string, metadata json.RawMessage) *apierror.APIError
	// SoftDelete tombstones a message (clears body/preview, sets deleted_at).
	SoftDelete(ctx context.Context, id, accountID string) *apierror.APIError

	// --- Customer-reply drafts (status=draft message rows) ---

	// CreateDraft inserts a customer-reply draft message (status='draft', no timeline sequence).
	CreateDraft(ctx context.Context, m *Message) *apierror.APIError
	// ListDrafts lists a conversation's draft-lifecycle messages (optionally filtered to one status), newest first.
	ListDrafts(ctx context.Context, conversationID, accountID string, status *string) ([]*Message, *apierror.APIError)
	// UpdateDraftContent edits a still-open draft's body/subject/preview (guarded on status='draft').
	UpdateDraftContent(ctx context.Context, id, accountID string, body string, subject, preview *string) *apierror.APIError
	// SetDraftStatus transitions an open draft (e.g. draft->rejected); applied is false if it was not open.
	SetDraftStatus(ctx context.Context, id, accountID, status string) (applied bool, apiErr *apierror.APIError)
	// SupersedeDraftsForThread marks a thread's still-open drafts superseded.
	SupersedeDraftsForThread(ctx context.Context, conversationID, sourceThreadMessageID string) *apierror.APIError
	// PromoteDraft promotes an open draft to a sent customer-visible timeline message in place: assigns
	// the sequence, sets visibility=external + kind + approver. Guarded on status='draft'
	// (compare-and-set); applied is false when a concurrent approve already sent it.
	PromoteDraft(ctx context.Context, id, accountID, kind string, sequence int64, approvedByAccountUserID, preview *string) (applied bool, apiErr *apierror.APIError)

	// --- Scheduled messages (status=scheduled message rows) ---

	// CreateScheduled inserts a future-delivery message (status='scheduled', no timeline sequence).
	CreateScheduled(ctx context.Context, m *Message) *apierror.APIError
	// ListScheduledByConversation lists the caller's scheduled messages in one conversation (soonest first).
	ListScheduledByConversation(ctx context.Context, conversationID, accountID, accountUserID string, limit int32) ([]*Message, *apierror.APIError)
	// CancelScheduled marks a scheduled message canceled if owned by the caller's account_user; canceled
	// is false when it is no longer scheduled, not owned, or not found.
	CancelScheduled(ctx context.Context, id, accountID, accountUserID string) (canceled bool, apiErr *apierror.APIError)
	// ListDueScheduled returns unclaimed scheduled messages whose delivery time has arrived.
	ListDueScheduled(ctx context.Context, limit int32) ([]*Message, *apierror.APIError)
	// ClaimScheduled claims a due scheduled message for delivery (compare-and-set on locked_at IS NULL);
	// claimed is false when another worker already claimed it.
	ClaimScheduled(ctx context.Context, id, lockOwner string) (claimed bool, apiErr *apierror.APIError)
	// PromoteScheduled promotes a claimed scheduled message to a sent timeline message in place (assigns
	// the sequence). Guarded on status='scheduled' (compare-and-set); applied is false if already delivered.
	PromoteScheduled(ctx context.Context, id string, sequence int64) (applied bool, apiErr *apierror.APIError)
	// MarkScheduledFailed records a delivery failure: bumps attempts, releases the claim, and sets the
	// new status ('scheduled' to retry, or terminal 'failed'/'canceled').
	MarkScheduledFailed(ctx context.Context, id, status string, lastError *string) *apierror.APIError
}

// BlockRepo persists 1:1 messaging blocks (one account_user blocking another).
// MessageReportRepo persists abuse reports filed against conversations/messages.
type MessageReportRepo interface {
	// Create inserts a report row.
	Create(ctx context.Context, report *MessageReport) *apierror.APIError
}

type BlockRepo interface {
	Create(ctx context.Context, id, accountID, blockerAccountUserID, blockedAccountUserID string) (*MessageBlock, *apierror.APIError)
	Delete(ctx context.Context, blockerAccountUserID, blockedAccountUserID string) *apierror.APIError
	// ExistsBetween reports whether either user has blocked the other.
	ExistsBetween(ctx context.Context, a, b string) (bool, *apierror.APIError)
	List(ctx context.Context, blockerAccountUserID string) ([]*MessageBlock, *apierror.APIError)
}

// MessageAttachmentRepo persists message attachments and loads them for message presentation.
type MessageAttachmentRepo interface {
	// Create inserts an attachment row (call within the send transaction).
	Create(ctx context.Context, a *MessageAttachment) *apierror.APIError
	// ListByMessageIDs batch-loads non-deleted attachments for the given messages.
	ListByMessageIDs(ctx context.Context, messageIDs []string) ([]*MessageAttachment, *apierror.APIError)
	// ListByConversation returns all live attachments across a conversation's messages (for redaction).
	ListByConversation(ctx context.Context, conversationID string) ([]*MessageAttachment, *apierror.APIError)
	// DeleteByID removes an attachment row after its S3 object has been deleted.
	DeleteByID(ctx context.Context, id string) *apierror.APIError
}

// NotificationPreferenceRepo persists per-(account_user, category) channel preferences. The effective preference resolves a category-specific row over the global (empty-category) default.
type NotificationPreferenceRepo interface {
	List(ctx context.Context, accountUserID string) ([]*NotificationPreference, *apierror.APIError)
	// GetEffective returns the resolved preference for a recipient + category, or not-found when neither a category-specific nor a global row exists (the caller applies channel defaults).
	GetEffective(ctx context.Context, accountUserID, category string) (*EffectiveNotificationPreference, *apierror.APIError)
	// Upsert creates or replaces the preference row for (account_user, category).
	Upsert(ctx context.Context, id, accountID, accountUserID string, input *UpsertNotificationPreferenceInput) *apierror.APIError
	// GetByUserCategory returns the stored row for (account_user, category), or not-found.
	GetByUserCategory(ctx context.Context, accountUserID, category string) (*NotificationPreference, *apierror.APIError)
}

// DeletedRecordRepo snapshots a row into deleted_record before a hard delete so the record is recoverable and repeat/racing deletes are distinguishable from "never existed".
type DeletedRecordRepo interface {
	Create(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID string, data any) *apierror.APIError
	Exists(ctx context.Context, resourceType constants.DeletedRecordResourceType, resourceID string) (bool, *apierror.APIError)
}

// IdempotencyKeyRepo persists idempotency rows for handlers that opt into contracts idempotency. Instances are constructed via RepoFactory.NewIdempotencyKeyRepo(); no transport layer wires it yet.
type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (string, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
