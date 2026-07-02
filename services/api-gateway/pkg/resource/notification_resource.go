package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleNotificationID = "nf_01h9z8q1w2e3r4t5y6u7i8o9"

// An in-app notification in the user's bell feed.
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
	Category constants.NotificationCategory `json:"category" validate:"required"`
	// Human-readable title.
	Title string `json:"title" validate:"required"`
	// Preview/body text.
	Body *string `json:"body"`
	// Where the notification is in its lifecycle.
	//
	// - `unseen`: not yet surfaced in the notification dropdown.
	// - `seen`: surfaced in the dropdown but not yet opened.
	// - `read`: explicitly opened by the user.
	// - `dismissed`: removed from the active feed.
	Status constants.NotificationStatus `json:"status" validate:"required"`
	// Delivery priority.
	Priority constants.NotificationPriority `json:"priority" validate:"required"`
	// The actor that generated this notification — a user, group, agent, or API key.
	//
	// System-generated notifications have no sender.
	Sender *Actor `json:"sender" expandable:"true"`
	// The app resource this notification links to, if any.
	Resource *Entity `json:"resource" expandable:"true"`
	// When the notification first appeared in the dropdown.
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

// NotificationUnreadCount summarizes a user's unread tallies across surfaces.
type NotificationUnreadCount struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_unread_count"`
	// Number of unseen bell notifications.
	Notifications int64 `json:"notifications"`
	// Number of conversations with unread messages (0 until chat ships).
	Conversations int64 `json:"conversations"`
	// Combined unread total.
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

// NotificationUnreadSummaryAccount is one account's unread tally in the cross-account summary.
type NotificationUnreadSummaryAccount struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_unread_summary_account"`
	// The account this tally is for.
	Account *Entity `json:"account" validate:"required"`
	// Number of unread items (notifications + announcements) in this account.
	Unread int64 `json:"unread"`
}

// NotificationUnreadSummary is the caller's unread totals across every account they belong to.
type NotificationUnreadSummary struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_unread_summary"`
	// Combined unread total across all of the caller's accounts.
	Total int64 `json:"total"`
	// Per-account unread tallies.
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

// NotificationSendResult acknowledges a notification send/fan-out request.
type NotificationSendResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_send_result"`
	// Number of recipients the notification was enqueued for.
	Enqueued int64 `json:"enqueued"`
}

var SampleNotificationSendResult = &NotificationSendResult{
	Object:   constants.ObjectTypeNotificationSendResult,
	Enqueued: 1,
}

func (*NotificationSendResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNotificationSendResult)
}
