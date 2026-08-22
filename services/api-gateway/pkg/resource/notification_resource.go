package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleNotificationID = "nf_yvw2bfj2guyn"

// An in-app notification addressed to a single user, shown in their notification (bell) feed.
//
// A notification belongs to one user in one account, so the feed you read is always that of the authenticated caller in the account they are acting in. Announcements broadcast to a whole account are a separate resource.
type Notification struct {
	// Notification ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification"`
	// The kind of event this notification represents.
	//
	// The set is open-ended and may grow over time. Common first-party categories are:
	//
	// - `chat.message`: a new message in a conversation.
	// - `chat.mention`: a direct @mention, delivered even when the conversation is muted.
	// - `chat.added`: the user was added to a conversation.
	// - `order.updated`: an order the user is involved with changed.
	// - `agent.run_completed`: an agent run the user triggered finished.
	// - `agent.alert`: an agent raised an alert during a run.
	// - `system.broadcast`: a targeted system message.
	// - `customer.registered`: a buyer completed registration on your customer portal.
	Category constants.NotificationCategory `json:"category" validate:"required"`
	// Short headline shown in the feed.
	Title string `json:"title" validate:"required"`
	// Supporting detail shown beneath the title, such as a preview of the message that triggered the notification.
	Body *string `json:"body"`
	// Where the notification is in its lifecycle.
	//
	// - `unseen`: delivered but not yet surfaced to the user.
	// - `seen`: surfaced in the feed but not yet opened.
	// - `read`: explicitly opened by the user.
	// - `dismissed`: removed from the active feed.
	//
	// The status is derived from the seen, read, and dismissed timestamps, and only ever moves forward — a notification can never become unseen again.
	Status constants.NotificationStatus `json:"status" validate:"required"`
	// How prominently the notification should be surfaced, from `low` through `urgent`.
	Priority constants.NotificationPriority `json:"priority" validate:"required"`
	// The actor that generated this notification.
	//
	// Notifications raised by the platform itself, rather than by a person, agent, or API key, have no sender.
	Sender *Actor `json:"sender" expandable:"true"`
	// The resource this notification is about, which the client can link to.
	//
	// Chat notifications point at the conversation the message was posted in — or at the support case, for customer-facing threads — so opening the notification opens the thread.
	Resource *Entity `json:"resource" expandable:"true"`
	// When the notification was first surfaced to the user.
	SeenAt *time.Time `json:"seen_at"`
	// When the notification was explicitly opened.
	ReadAt *time.Time `json:"read_at"`
	// When the notification was dismissed.
	DismissedAt *time.Time `json:"dismissed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleNotification = &Notification{
	ID:        SampleNotificationID,
	Object:    constants.ObjectTypeNotification,
	Category:  constants.NotificationCategoryOrderUpdated,
	Status:    constants.NotificationStatusUnseen,
	Title:     "Order updated",
	Body:      new("Order #1024 changed from estimate to confirmed."),
	Sender:    NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Jie Yan"), nil),
	Resource:  NewEntity(SampleSalesOrderID, constants.ObjectTypeSalesOrder, new(SampleSalesOrderNumber), nil),
	Priority:  constants.NotificationPriorityNormal,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
