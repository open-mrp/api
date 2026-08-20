package constants

import "time"

// Resolves the start of the account's billing period — a month back from the subscription's period
// end, or the first of the calendar month. Shared so per-period caps and usage counts agree.
func BillingPeriodStart(subscriptionPeriodEnd *time.Time) time.Time {
	if subscriptionPeriodEnd != nil {
		return subscriptionPeriodEnd.AddDate(0, -1, 0)
	}
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}
