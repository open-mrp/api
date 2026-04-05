package constants

// SubscriptionStatus represents the status of a Stripe subscription.
type SubscriptionStatus string

const (
	// SubscriptionStatusActive indicates the subscription is current and fully paid.
	SubscriptionStatusActive SubscriptionStatus = "active"
	// SubscriptionStatusTrialing indicates the subscription is in a free trial period.
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
	// SubscriptionStatusPastDue indicates the subscription has an outstanding unpaid invoice.
	SubscriptionStatusPastDue SubscriptionStatus = "past_due"
	// SubscriptionStatusCanceled indicates the subscription has been canceled.
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	// SubscriptionStatusUnpaid indicates the subscription is unpaid after exhausting retry attempts.
	SubscriptionStatusUnpaid SubscriptionStatus = "unpaid"
)

func (s SubscriptionStatus) String() string {
	return string(s)
}

func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case SubscriptionStatusActive, SubscriptionStatusTrialing, SubscriptionStatusPastDue, SubscriptionStatusCanceled, SubscriptionStatusUnpaid:
		return true
	default:
		return false
	}
}

func (s SubscriptionStatus) EnumValues() []string {
	return []string{string(SubscriptionStatusActive), string(SubscriptionStatusTrialing), string(SubscriptionStatusPastDue), string(SubscriptionStatusCanceled), string(SubscriptionStatusUnpaid)}
}
