package domain

import (
	"encoding/json"
	"time"
)

type EmailLog struct {
	ID           string
	HasSent      bool
	AccountID    string
	SentByID     *string
	Subject      *string
	Filename     *string
	SesMessageID *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Notification is a per-user bell-feed entry. It references a message or stands alone (an event/system alert). The bell feed is a projection over these rows.
type Notification struct {
	ID                     string
	AccountID              string
	RecipientAccountUserID string
	Category               string
	SourceMessageID        *string
	ConversationID         *string
	Title                  string
	Body                   *string
	TemplateKey            *string
	TemplateParams         json.RawMessage
	LinkResourceType       *string
	LinkResourceID         *string
	// Polymorphic sender attribution (user | group | system | agent | apikey).
	SenderType  *string
	SenderID    *string
	SenderName  *string
	Priority    string
	SeenAt      *time.Time
	ReadAt      *time.Time
	DismissedAt *time.Time
	Metadata    json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NotificationListFilter parameterizes a keyset-paginated feed query. A nil filter pointer means "no constraint"; Seen/Read filter on presence of the respective timestamp. The cursor is (CursorCreatedAt, CursorID); both nil means the first page.
type NotificationListFilter struct {
	RecipientAccountUserID string
	Category               *string
	// Status filters by lifecycle state ("unseen"/"seen"/"read"/"dismissed"); nil = the default active feed (all non-dismissed).
	Status *string
	// Search is a free-text term matched against title/body (LIKE). nil = no search.
	Search *string
	// SenderIDs / SenderTypes filter by sender attribution (multi-value, IN). Empty = no filter.
	SenderIDs       []string
	SenderTypes     []string
	Limit           int32
	CursorCreatedAt *time.Time
	CursorID        *string
}

// UnreadCounts is the caller's unread tally across surfaces. In Phase 1, Conversations is always 0 (chat ships later); Total = Notifications + Conversations.
type UnreadCounts struct {
	Notifications int64
	Conversations int64
	Total         int64
}

// AccountUnread is a per-account unread tally for the cross-account summary.
type AccountUnread struct {
	AccountID string
	Unread    int64
}

// UnreadSummary is the caller's unread totals across every account they belong to. It powers the cross-account bell hint (a dot while viewing a different account).
type UnreadSummary struct {
	Total    int64
	Accounts []AccountUnread
}

// Announcement is a broadcast notification stored once (platform- or account-scoped) with per-user read state tracked separately in announcement_receipt. The Seen/Read/Dismissed fields are the caller-specific receipt state, joined in at read time.
type Announcement struct {
	ID               string
	Scope            string
	AccountID        *string
	Category         string
	TemplateKey      *string
	TemplateParams   json.RawMessage
	Title            string
	Body             *string
	LinkResourceType *string
	LinkResourceID   *string
	Priority         string
	PublishAt        time.Time
	ExpiresAt        *time.Time
	CreatedBy        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	// Caller-specific receipt state (nil when the caller has no receipt yet).
	SeenAt      *time.Time
	ReadAt      *time.Time
	DismissedAt *time.Time
}

// CreateAnnouncementInput is the validated input for creating a broadcast announcement.
type CreateAnnouncementInput struct {
	Scope            string
	AccountID        *string
	Category         string
	Title            string
	Body             *string
	TemplateKey      *string
	TemplateParams   json.RawMessage
	LinkResourceType *string
	LinkResourceID   *string
	Priority         string
	PublishAt        time.Time
	ExpiresAt        *time.Time
	CreatedBy        *string
}

// AnnouncementListFilter parameterizes the active-announcement feed for one caller. The cursor is (CursorPublishAt, CursorID); both nil means the first page.
type AnnouncementListFilter struct {
	AccountUserID   string
	AccountID       *string
	Limit           int32
	CursorPublishAt *time.Time
	CursorID        *string
}

// AnnouncementPage is one page of active announcements plus the next opaque cursor.
type AnnouncementPage struct {
	Announcements []*Announcement
	NextCursor    *string
	HasNextPage   bool
}

type IdempotencyKey struct {
	ID             int64
	TypeID         string
	ServiceName    string
	Handler        string
	IdempotencyKey string
	ActorID        *string
	IdentityType   string
	ScopeHash      string
	ResponseCode   *int
	ResponseBody   json.RawMessage
	RecoveryPoint  string
}

func (k *IdempotencyKey) HasResponse() bool {
	return k.ResponseCode != nil
}

func (k *IdempotencyKey) IsFinished() bool {
	return k.RecoveryPoint == string(RecoveryPointFinished)
}
