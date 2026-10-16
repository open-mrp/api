package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleNotificationID = "nf_yvw2bfj2guyn"

// Something that happened that you should know about, shown in your notification feed.
//
// Your feed holds only your own notifications in the account you're acting in. Where notifications also go by email or push follows your notification preferences.
type Notification struct {
	// Notification ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification"`
	// What happened.
	//
	// - `chat.message`: a new message in a conversation you're in.
	// - `chat.mention`: you were @mentioned in a conversation, even one you muted.
	// - `chat.added`: you were added to a conversation.
	// - `note.mention`: you were @mentioned in a note.
	// - `order.updated`: a sales order you follow reached a milestone.
	// - `production_run.updated`: batches were added to or removed from a production run you're responsible for.
	// - `invoice.payment_status_changed`: an invoice you marked paid by hand was recalculated.
	// - `customer.registered`: a buyer registered on your customer portal.
	// - `agent.run_completed`: an agent run you started finished.
	// - `agent.alert`: an agent raised an alert.
	// - `system.broadcast`: a message from your team or OpenMRP.
	// - `security.sign_in`, `security.credentials_changed`, `security.device_paired`: account security events. These always email you.
	Type constants.NotificationCategory `json:"type" validate:"required"`
	// Short headline shown in the feed.
	Title string `json:"title" validate:"required"`
	// More detail, such as a preview of the message or what changed.
	Body *string `json:"body"`
	// Where the notification is in its lifecycle: `unseen`, `seen` (shown in your feed), `read` (opened) or `dismissed` (cleared from your feed).
	//
	// A notification only moves forward through these states.
	Status constants.NotificationStatus `json:"status" validate:"required"`
	// How prominently to show the notification: `low`, `normal`, `high` or `urgent`.
	Priority constants.NotificationPriority `json:"priority" validate:"required"`
	// Who caused the notification. Null for notifications OpenMRP raises itself.
	Sender *Actor `json:"sender" expandable:"true"`
	// The record the notification is about, such as the order that changed or the conversation with a new message.
	Related *NotificationRelated `json:"related"`
	// How many events this notification stands for.
	//
	// Several milestones on one record in a day share one notification, which counts them here; most notifications stand for one.
	Occurrences int64 `json:"occurrences" validate:"required"`
	// When the notification was first shown in your feed.
	SeenAt *time.Time `json:"seen_at"`
	// When you opened the notification.
	ReadAt *time.Time `json:"read_at"`
	// When you dismissed the notification.
	DismissedAt *time.Time `json:"dismissed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
	// Resource is preview.9's link; forge reads related.record.
	Resource *Entity `json:"resource" expandable:"true" visibility:"internal"`
}

// The record a notification is about.
type NotificationRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_related"`
	// The record the notification is about, which a client can link to.
	Record *Record `json:"record" expandable:"true"`
}

var SampleNotification = &Notification{
	ID:       SampleNotificationID,
	Object:   constants.ObjectTypeNotification,
	Type:     constants.NotificationCategoryOrderUpdated,
	Status:   constants.NotificationStatusUnseen,
	Title:    "Sales order 1024 updated",
	Body:     new("Jie Yan issued the order."),
	Sender:   NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Jie Yan"), nil),
	Priority: constants.NotificationPriorityNormal,
	Related: &NotificationRelated{
		Object: constants.ObjectTypeNotificationRelated,
		Record: &Record{ID: SampleSalesOrderID, Object: constants.ObjectTypeRecord, Type: constants.RecordTypeSalesOrder, Number: new(SampleSalesOrderNumber), Status: new("issued"), Metadata: map[string]string{}},
	},
	Occurrences: 2,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Notification) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNotification)
}

// The caller's unread tallies in one account, used to drive the notification bell badge.
type NotificationUnreadCount struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_unread_count"`
	// Number of the caller's notifications that have not been seen yet.
	//
	// Dismissed notifications are never counted, and marking all notifications seen drops this to zero.
	Notifications int64 `json:"notifications"`
	// Number of conversations with unread messages.
	//
	// Always `0` today — conversation unread counts are not yet folded into the bell.
	Conversations int64 `json:"conversations"`
	// Combined unread total for the bell badge.
	//
	// This is the unseen notification count plus any account announcements the caller has not seen, so it can exceed `notifications`. Announcements are cleared individually rather than by marking all notifications seen.
	Total int64 `json:"total"`
}

var SampleNotificationUnreadCount = &NotificationUnreadCount{
	Object:        constants.ObjectTypeNotificationUnreadCount,
	Notifications: 3,
	Conversations: 0,
	Total:         3,
}

func (*NotificationUnreadCount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNotificationUnreadCount)
}

// One account's unread tally within the caller's cross-account summary.
type NotificationUnreadSummaryAccount struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_unread_summary_account"`
	// The account this tally is for.
	Account *Entity `json:"account" validate:"required"`
	// Number of unseen notifications and account announcements the caller has in this account.
	Unread int64 `json:"unread"`
}

// The caller's unread totals across every account they belong to, used to show unread activity waiting in accounts they are not currently working in.
type NotificationUnreadSummary struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_unread_summary"`
	// Combined unread total across all of the caller's accounts.
	Total int64 `json:"total"`
	// Per-account unread tallies.
	//
	// Every account the caller belongs to is listed, including accounts with nothing unread.
	Accounts *List[NotificationUnreadSummaryAccount] `json:"accounts"`
}

var SampleNotificationUnreadSummary = &NotificationUnreadSummary{
	Object: constants.ObjectTypeNotificationUnreadSummary,
	Total:  5,
	Accounts: NewList([]NotificationUnreadSummaryAccount{
		{Object: constants.ObjectTypeNotificationUnreadSummaryAccount, Account: NewEntity(SampleAccountID, constants.ObjectTypeAccount, nil, nil), Unread: 5},
	}, PageInfo{}),
}

func (*NotificationUnreadSummary) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNotificationUnreadSummary)
}

// The acknowledgement returned when a notification is accepted for delivery.
type NotificationSendResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_send_result"`
	// Number of deliveries accepted for the notification.
	//
	// An account broadcast is stored once as a single announcement that serves everyone in the account, so it reports `1` rather than a per-user count. Acceptance is not delivery: recipients who cannot be resolved are skipped when the notification is fanned out.
	Enqueued int64 `json:"enqueued"`
}

var SampleNotificationSendResult = &NotificationSendResult{
	Object:   constants.ObjectTypeNotificationSendResult,
	Enqueued: 1,
}

func (*NotificationSendResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNotificationSendResult)
}
