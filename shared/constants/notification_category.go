package constants

// NotificationCategory is a notification's type (`type` in the API; stored as `category`). The set is fixed: a new
// producer adds its value here.
type NotificationCategory string

const (
	// NotificationCategoryChatMessage is a new chat message in a conversation.
	NotificationCategoryChatMessage NotificationCategory = "chat.message"
	// NotificationCategoryChatMention is a direct @mention (pierces mute).
	NotificationCategoryChatMention NotificationCategory = "chat.mention"
	// NotificationCategoryChatAdded indicates the user was added to a conversation.
	NotificationCategoryChatAdded NotificationCategory = "chat.added"
	// NotificationCategoryOrderUpdated indicates a change to an order the user is involved with.
	NotificationCategoryOrderUpdated NotificationCategory = "order.updated"
	// NotificationCategoryAgentRunCompleted indicates an agent run the user triggered finished.
	NotificationCategoryAgentRunCompleted NotificationCategory = "agent.run_completed"
	// NotificationCategoryAgentAlert is an alert an agent raised during a run.
	NotificationCategoryAgentAlert NotificationCategory = "agent.alert"
	// NotificationCategorySystemBroadcast is a targeted system message to a user.
	NotificationCategorySystemBroadcast NotificationCategory = "system.broadcast"
	// NotificationCategoryCustomerRegistered indicates a buyer completed registration on the account's customer portal; sent to the customer-service support group.
	NotificationCategoryCustomerRegistered NotificationCategory = "customer.registered"
	// NotificationCategoryProductionRunUpdated indicates someone other than a production run's responsible user added or deleted its batches.
	NotificationCategoryProductionRunUpdated NotificationCategory = "production_run.updated"
	// NotificationCategoryInvoicePaymentStatusChanged tells the person who set an invoice's paid-in-full flag by hand that recalculating its payments overturned it.
	NotificationCategoryInvoicePaymentStatusChanged NotificationCategory = "invoice.payment_status_changed"
	// NotificationCategoryNoteMention tells a person they were @mentioned in a note.
	NotificationCategoryNoteMention NotificationCategory = "note.mention"
	// NotificationCategorySecuritySignIn reports a new sign-in to the person's account. Required.
	NotificationCategorySecuritySignIn NotificationCategory = "security.sign_in"
	// NotificationCategorySecurityCredentialsChanged reports a passkey or two-factor change. Required.
	NotificationCategorySecurityCredentialsChanged NotificationCategory = "security.credentials_changed"
	// NotificationCategorySecurityDevicePaired reports a device paired to the person. Required.
	NotificationCategorySecurityDevicePaired NotificationCategory = "security.device_paired"
)

// Required reports whether the type always emails and can't be turned off.
func (c NotificationCategory) Required() bool {
	switch c {
	case NotificationCategorySecuritySignIn, NotificationCategorySecurityCredentialsChanged, NotificationCategorySecurityDevicePaired:
		return true
	}
	return false
}

func (c NotificationCategory) IsValid() bool {
	switch c {
	case NotificationCategoryChatMessage,
		NotificationCategoryChatMention,
		NotificationCategoryChatAdded,
		NotificationCategoryOrderUpdated,
		NotificationCategoryAgentRunCompleted,
		NotificationCategoryAgentAlert,
		NotificationCategorySystemBroadcast,
		NotificationCategoryCustomerRegistered,
		NotificationCategoryProductionRunUpdated,
		NotificationCategoryInvoicePaymentStatusChanged,
		NotificationCategoryNoteMention,
		NotificationCategorySecuritySignIn,
		NotificationCategorySecurityCredentialsChanged,
		NotificationCategorySecurityDevicePaired:
		return true
	default:
		return false
	}
}

func (c NotificationCategory) EnumValues() []string {
	return []string{
		string(NotificationCategoryChatMessage),
		string(NotificationCategoryChatMention),
		string(NotificationCategoryChatAdded),
		string(NotificationCategoryOrderUpdated),
		string(NotificationCategoryAgentRunCompleted),
		string(NotificationCategoryAgentAlert),
		string(NotificationCategorySystemBroadcast),
		string(NotificationCategoryCustomerRegistered),
		string(NotificationCategoryProductionRunUpdated),
		string(NotificationCategoryInvoicePaymentStatusChanged),
		string(NotificationCategoryNoteMention),
		string(NotificationCategorySecuritySignIn),
		string(NotificationCategorySecurityCredentialsChanged),
		string(NotificationCategorySecurityDevicePaired),
	}
}

func (c *NotificationCategory) StringPtr() *string {
	if c == nil {
		return nil
	}
	v := string(*c)
	return &v
}
