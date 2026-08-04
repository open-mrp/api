package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

const SampleStripeCustomerID = "cus_OG9R5zKr5xJHHp"
const SampleBillingPortalURL = "https://billing.stripe.com/p/session/test_YWNjdF8xTTJKVGtMa3E0Z3Bic"

// A usage metric with its current value and any applicable limit.
type UsageItem struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=usage_item"`
	// Current usage count.
	Current int `json:"current"`
	// Maximum allowed usage.
	//
	// Null means unlimited.
	Limit *int `json:"limit"`
}

// Account usage metrics across all resource types.
//
// Per-period counts are measured from the start of the account's current billing period, which falls back to the start of the calendar month when the account has no active subscription.
type AccountUsageResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_usage_response"`
	// Seat usage: users on the account counted against the plan's seat limit.
	Seats UsageItem `json:"seats" validate:"required"`
	// Invoice usage, counted within the current billing period.
	Invoices UsageItem `json:"invoices" validate:"required"`
	// Batch usage, counted within the current billing period.
	Batches UsageItem `json:"batches" validate:"required"`
	// Sandbox usage: sandbox environments on the account counted against the plan's sandbox limit.
	Sandboxes UsageItem `json:"sandboxes" validate:"required"`
	// Status of the account's billing subscription.
	Subscription *SubscriptionInfo `json:"subscription"`
	// Estimated agent LLM spending for the current billing month, and the cap it is measured against.
	AgentSpend *AgentSpendInfo `json:"agent_spend"`
	// Display name of the plan the account is actually billed on, resolved live from Stripe (e.g. `Founder`).
	//
	// Empty when the account has no Stripe pricing plan.
	PlanName string `json:"plan_name"`
	// Flat base fee in cents charged each `base_fee_interval`, resolved live from Stripe.
	//
	// `0` when the plan is priced per seat rather than a flat base fee.
	BaseFeeCents int64 `json:"base_fee_cents"`
	// Interval the base fee recurs on (e.g. `month`).
	//
	// Empty when there is no base fee.
	BaseFeeInterval string `json:"base_fee_interval"`
}

// Subscription status information.
type SubscriptionInfo struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=subscription_info"`
	// Whether the subscription is actively being serviced.
	//
	// Typically one of:
	// - `active`: the subscription is in good standing.
	// - `paused`: servicing is temporarily suspended.
	// - `canceled`: the subscription has been canceled.
	ServicingStatus string `json:"servicing_status" validate:"required"`
	// Payment collection status of the subscription.
	//
	// Typically one of:
	// - `current`: payments are being collected normally.
	// - `paused`: payment collection is temporarily suspended.
	// - `awaiting_customer_action`: a payment requires action from the customer (e.g., updating a payment method).
	CollectionStatus string `json:"collection_status" validate:"required"`
}

// A short-lived link into the Stripe billing portal, where an account admin can manage payment methods, invoices, and the subscription.
type BillingPortalSessionResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=billing_portal_session_response"`
	// URL to send the admin to.
	//
	// The link is issued by Stripe for a single visit and expires; generate a new session each time. On leaving the portal the admin is returned to the dashboard's billing page.
	URL string `json:"url" validate:"required"`
}

// Result of a plan switch.
type SwitchPlanResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=switch_plan_response"`
	// Whether the plan switch was applied successfully.
	Success bool `json:"success"`
	// ID of the Stripe billing intent that was committed to apply the change.
	//
	// Returned when switching to a paid plan; absent when switching to the free plan.
	IntentID *string `json:"intent_id"`
}

// Result of ensuring a billing customer exists.
type EnsureBillingCustomerResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=ensure_billing_customer_response"`
	// Stripe customer ID.
	StripeCustomerID string `json:"stripe_customer_id" validate:"required"`
	// Whether a new Stripe customer was created by this call.
	//
	// `false` means the account already had a Stripe customer, which was returned instead.
	Created bool `json:"created"`
	// ID of the account's Stripe billing profile.
	//
	// The billing profile and its billing cadence are set up when the account is first prepared for paid billing, not by creating the Stripe customer.
	BillingProfileID *string `json:"billing_profile_id"`
}

// Monthly agent spending cap for an account.
type SpendingCapResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=spending_cap_response"`
	// Ceiling in cents on estimated agent spending per billing month.
	//
	// Null means agent spending is uncapped.
	CapCents *int64 `json:"cap_cents"`
}

// Estimated agent LLM spending for the current billing month.
type AgentSpendInfo struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_spend_info"`
	// Estimated spend in cents for the current billing month.
	//
	// Priced at the same token rates the account is billed at, and cached briefly, so it can trail live usage by a short interval.
	EstimatedSpendCents int64 `json:"estimated_spend_cents"`
	// Ceiling in cents on estimated agent spending per billing month.
	//
	// Null means agent spending is uncapped.
	CapCents *int64 `json:"cap_cents"`
}

// Acknowledgement that a webhook event was accepted.
type WebhookResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=webhook_response"`
	// Whether the event was accepted for processing.
	//
	// Acceptance means the signature was verified and the event was handled or queued. Event types Augno takes no action on are acknowledged the same way, so this is not a signal that anything changed.
	Received bool `json:"received"`
}

// Request for Stripe webhook processing.
type StripeWebhookRequest struct {
	// Raw request body bytes for signature verification.
	RawBody []byte `rawbody:"true"`
	// Stripe-Signature header value for payload verification.
	Signature string `header:"Stripe-Signature"`
}

var SampleUsageItem = &UsageItem{
	Object:  constants.ObjectTypeUsageItem,
	Current: 5,
	Limit:   new(10),
}

var SampleUsageItemUnlimited = &UsageItem{
	Object:  constants.ObjectTypeUsageItem,
	Current: 100,
	Limit:   nil,
}

var SampleSubscriptionInfo = &SubscriptionInfo{
	Object:           constants.ObjectTypeSubscriptionInfo,
	ServicingStatus:  "active",
	CollectionStatus: "current",
}

var SampleAgentSpendInfo = &AgentSpendInfo{
	Object:              constants.ObjectTypeAgentSpendInfo,
	EstimatedSpendCents: 1875,
	CapCents:            &sampleCapCents,
}

var SampleAccountUsageResponse = &AccountUsageResponse{
	Object:          constants.ObjectTypeAccountUsageResponse,
	Seats:           *SampleUsageItem,
	Invoices:        *SampleUsageItemUnlimited,
	Batches:         *SampleUsageItemUnlimited,
	Sandboxes:       *SampleUsageItem,
	Subscription:    SampleSubscriptionInfo,
	AgentSpend:      SampleAgentSpendInfo,
	PlanName:        "Founder",
	BaseFeeCents:    100,
	BaseFeeInterval: "month",
}

var SampleBillingPortalSessionResponse = &BillingPortalSessionResponse{
	Object: constants.ObjectTypeBillingPortalSessionResponse,
	URL:    SampleBillingPortalURL,
}

var sampleSwitchPlanIntentID = "bintent_1MtHb3LkdIwHu7ixxOzzPQ12"

var SampleSwitchPlanResponse = &SwitchPlanResponse{
	Object:   constants.ObjectTypeSwitchPlanResponse,
	Success:  true,
	IntentID: &sampleSwitchPlanIntentID,
}

var sampleBillingProfileID = "bpr_1OG9R5zKr5xJHHpQ8Zk"

var SampleEnsureBillingCustomerResponse = &EnsureBillingCustomerResponse{
	Object:           constants.ObjectTypeEnsureBillingCustomerResponse,
	StripeCustomerID: SampleStripeCustomerID,
	Created:          true,
	BillingProfileID: &sampleBillingProfileID,
}

var SampleWebhookResponse = &WebhookResponse{
	Object:   constants.ObjectTypeWebhookResponse,
	Received: true,
}

func (*UsageItem) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUsageItem)
}

func (*AccountUsageResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountUsageResponse)
}

func (*SubscriptionInfo) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSubscriptionInfo)
}

func (*BillingPortalSessionResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleBillingPortalSessionResponse)
}

func (*SwitchPlanResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSwitchPlanResponse)
}

func (*EnsureBillingCustomerResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEnsureBillingCustomerResponse)
}

func (*WebhookResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleWebhookResponse)
}

var sampleCapCents int64 = 5000

var SampleSpendingCapResponse = &SpendingCapResponse{
	Object:   constants.ObjectTypeSpendingCapResponse,
	CapCents: &sampleCapCents,
}

func (*SpendingCapResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSpendingCapResponse)
}
