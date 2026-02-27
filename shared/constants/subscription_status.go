package constants

// SubscriptionStatus represents the status of a Stripe subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid   SubscriptionStatus = "unpaid"
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
