package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/notifyprefs"
)

// Subscription is whether one person is notified about one record.
type Subscription struct {
	ID            string
	AccountID     string
	AccountUserID string
	RecordType    string `audit:"record_type"`
	RecordID      string `audit:"record_id"`
	Reason        string `audit:"reason"`
	Status        string `audit:"status"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SubscriptionListFilter is one page of a person's subscriptions, newest first.
type SubscriptionListFilter struct {
	AccountUserID   string
	RecordIDs       []string
	RecordTypes     []string
	Statuses        []string
	Limit           int32
	CursorCreatedAt *time.Time
	CursorID        *string
}

// ListSubscriptionsInput is the caller's subscription list request.
type ListSubscriptionsInput struct {
	RecordIDs   []string
	RecordTypes []string
	Statuses    []string
	Limit       int32
	Cursor      *string
}

// SubscriptionPage is one page of subscriptions.
type SubscriptionPage struct {
	Subscriptions []*Subscription
	NextCursor    *string
	HasNextPage   bool
}

// NotificationSetting is a person's (AccountUserID set) or the account's (AccountUserID empty) settings that aren't per
// notification type.
type NotificationSetting struct {
	AccountID     string
	AccountUserID string
	// Defaults is the account's defaults; unused on a person's row.
	Defaults notifyprefs.Defaults
	// AutoSubscribe is a person's own auto-subscribe list; nil inherits.
	AutoSubscribe []constants.AutoSubscribeTrigger
	PausedUntil   *time.Time
}

// NotificationPreferences is a person's effective notification settings.
type NotificationPreferences struct {
	Default       notifyprefs.Setting
	Types         map[string]notifyprefs.Setting
	AutoSubscribe []constants.AutoSubscribeTrigger
	PausedUntil   *time.Time
}

// UpdateNotificationPreferencesInput changes a person's settings. PausedUntilSet with a nil PausedUntil resumes.
type UpdateNotificationPreferencesInput struct {
	Patch          notifyprefs.Patch
	PausedUntilSet bool
	PausedUntil    *time.Time
}
