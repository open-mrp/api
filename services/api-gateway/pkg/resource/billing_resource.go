package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/timeutil"
)

const SampleStripeCustomerID = "cus_OG9R5zKr5xJHHp"
const SampleSubscriptionID = "sub_1Qw4Rn2eZvKYlo2C0ghJ3kXa"
const SampleBillingPortalURL = "https://billing.stripe.com/p/session/test_YWNjdF8xTTJKVGtMa3E0Z3Bic"
const SampleCheckoutURL = "https://checkout.stripe.com/c/pay/cs_test_a1VnbGQ4ZTFRdGRqUWpYR3h6OG"
const SampleClientSecret = "cs_test_a1VnbGQ4ZTFRdGRqUWpYR3h6OG_secret_fk3Mds9RqwEm5J" // #nosec G101 - Test fixture, not a real credential
const SamplePublishableKey = "pk_test_51OjP6DKQnQhqLJrS7DG5gHN00xY3bRzWp"

var SampleUsageItem = &UsageItem{
	Current: 5,
	Limit:   new(10),
}

var SampleUsageItemUnlimited = &UsageItem{
	Current: 100,
	Limit:   nil,
}

var SampleSubscriptionInfo = &SubscriptionInfo{
	Status:            "trialing",
	CurrentPeriodEnd:  timeutil.TimestampToTimePtr(sampleTrialEndTimestamp),
	TrialEnd:          timeutil.TimestampToTimePtr(sampleTrialEndTimestamp),
	CancelAtPeriodEnd: false,
	CancelAt:          nil,
}

var SampleAccountUsageResponse = &AccountUsageResponse{
	Seats:        *SampleUsageItem,
	Invoices:     *SampleUsageItemUnlimited,
	Batches:      *SampleUsageItemUnlimited,
	Sandboxes:    *SampleUsageItem,
	Subscription: SampleSubscriptionInfo,
}

var SampleBillingPortalSessionResponse = &BillingPortalSessionResponse{
	URL: SampleBillingPortalURL,
}

var SampleSwitchPlanResponse = &SwitchPlanResponse{
	Success:         true,
	RequiresPayment: false,
	CheckoutURL:     new(SampleCheckoutURL),
}

var SampleConfirmPlanSwitchResponse = &ConfirmPlanSwitchResponse{
	Success: true,
}

var SampleEnsureBillingCustomerResponse = &EnsureBillingCustomerResponse{
	StripeCustomerID: SampleStripeCustomerID,
	Created:          true,
}

var SampleWebhookResponse = &WebhookResponse{
	Received: true,
}

// UsageItem represents a single usage metric with current value and optional limit.
type UsageItem struct {
	// The current usage count.
	Current int `json:"current"`
	// The maximum allowed usage, null means unlimited.
	Limit *int `json:"limit"`
}

func (*UsageItem) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUsageItem)
}

// AccountUsageResponse represents account usage metrics across all resource types.
type AccountUsageResponse struct {
	// The seat usage for the account.
	Seats UsageItem `json:"seats" validate:"required"`
	// The invoice usage for the account.
	Invoices UsageItem `json:"invoices" validate:"required"`
	// The batch usage for the account.
	Batches UsageItem `json:"batches" validate:"required"`
	// The sandbox usage for the account.
	Sandboxes UsageItem `json:"sandboxes" validate:"required"`
	// Subscription status information.
	Subscription *SubscriptionInfo `json:"subscription"`
}

func (*AccountUsageResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountUsageResponse)
}

// SubscriptionInfo represents subscription status and billing information.
type SubscriptionInfo struct {
	// The subscription status (active, trialing, past_due, canceled, etc.).
	Status string `json:"status" validate:"required"`
	// When the current billing period ends.
	CurrentPeriodEnd *time.Time `json:"current_period_end,omitempty"`
	// When the trial ends, if subscription is trialing.
	TrialEnd *time.Time `json:"trial_end,omitempty"`
	// Whether the subscription will cancel at the end of the current period.
	CancelAtPeriodEnd bool `json:"cancel_at_period_end"`
	// When the subscription will be canceled, if scheduled for cancellation.
	CancelAt *time.Time `json:"cancel_at,omitempty"`
}

func (*SubscriptionInfo) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSubscriptionInfo)
}

// BillingPortalSessionResponse represents a Stripe billing portal session.
type BillingPortalSessionResponse struct {
	// The URL to redirect the user to the Stripe billing portal.
	URL string `json:"url" validate:"required"`
}

func (*BillingPortalSessionResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleBillingPortalSessionResponse)
}

// SwitchPlanResponse represents the result of initiating a plan switch.
type SwitchPlanResponse struct {
	// Whether the plan switch was initiated successfully.
	Success bool `json:"success"`
	// Whether the plan switch requires payment via checkout.
	RequiresPayment bool `json:"requires_payment"`
	// The Stripe checkout URL, if payment is required.
	CheckoutURL *string `json:"checkout_url,omitempty"`
}

func (*SwitchPlanResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSwitchPlanResponse)
}

// ConfirmPlanSwitchResponse represents the result of confirming a plan switch.
type ConfirmPlanSwitchResponse struct {
	// Whether the plan switch was confirmed successfully.
	Success bool `json:"success"`
}

func (*ConfirmPlanSwitchResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConfirmPlanSwitchResponse)
}

// EnsureBillingCustomerResponse represents the result of ensuring a billing customer exists.
type EnsureBillingCustomerResponse struct {
	// The ID of the Stripe customer for billing.
	StripeCustomerID string `json:"stripe_customer_id" validate:"required"`
	// Indicates whether a new Stripe customer was created.
	Created bool `json:"created"`
}

func (*EnsureBillingCustomerResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEnsureBillingCustomerResponse)
}

// WebhookResponse represents the result of processing a webhook.
type WebhookResponse struct {
	// Whether the webhook was received and processed.
	Received bool `json:"received"`
}

func (*WebhookResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleWebhookResponse)
}

// StripeWebhookRequest carries the raw body and signature for Stripe webhook verification.
type StripeWebhookRequest struct {
	RawBody   []byte `rawbody:"true"`
	Signature string `header:"Stripe-Signature"`
}
