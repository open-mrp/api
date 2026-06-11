package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

const SampleStripeCustomerID = "cus_OG9R5zKr5xJHHp"
const SampleBillingPortalURL = "https://billing.stripe.com/p/session/test_YWNjdF8xTTJKVGtMa3E0Z3Bic"

// Usage metric with current value and optional limit.
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

// Detailed agent token usage breakdown.
type AgentTokenDetail struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_token_detail"`
	// Tokens included in the plan.
	IncludedTokens int64 `json:"included_tokens"`
	// Tokens used in the current billing period.
	UsedTokens int64 `json:"used_tokens"`
	// Input tokens used in the current billing period.
	InputTokens int64 `json:"input_tokens"`
	// Output tokens used in the current billing period.
	OutputTokens int64 `json:"output_tokens"`
	// Additional tokens purchased via token packs.
	AdditionalTokensPurchased int64 `json:"additional_tokens_purchased"`
	// Total tokens available (included + purchased).
	TotalAvailable int64 `json:"total_available"`
	// Estimated cost in dollars for the current billing period.
	//
	// This is a rough estimate for display purposes; the actual billed amount is calculated by the billing provider and may differ.
	CurrentPeriodCost float64 `json:"current_period_cost"`
	// When the current billing period ends (ISO 8601).
	BillingPeriodEnd string `json:"billing_period_end"`
	// Cost in dollars per million tokens for usage beyond the available token balance.
	OverageCostPerMillionTokens float64 `json:"overage_cost_per_million_tokens"`
}

// Account usage metrics across all resource types.
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
	// Subscription status information.
	//
	// Null if the account has no recorded subscription.
	Subscription *SubscriptionInfo `json:"subscription"`
	// Estimated agent LLM spending for the current month.
	AgentSpend *AgentSpendInfo `json:"agent_spend"`
	// Detailed agent token usage breakdown.
	AgentTokenDetail *AgentTokenDetail `json:"agent_token_detail"`
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

// Stripe billing portal session.
type BillingPortalSessionResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=billing_portal_session_response"`
	// Redirect URL for the Stripe billing portal.
	URL string `json:"url" validate:"required"`
}

// Result of initiating a plan switch.
type SwitchPlanResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=switch_plan_response"`
	// Whether the plan switch was initiated successfully.
	Success bool `json:"success"`
	// ID of the billing intent committed for the switch.
	//
	// Set to the committed billing intent ID for paid plan changes. Null for switches to the free plan, where no intent is surfaced.
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
	// Billing profile ID, if one was created.
	BillingProfileID *string `json:"billing_profile_id"`
}

// Monthly agent spending cap for an account.
type SpendingCapResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=spending_cap_response"`
	// Monthly spending cap in cents.
	//
	// Null means no cap.
	CapCents *int64 `json:"cap_cents"`
}

// Estimated agent LLM spending for the current billing month.
type AgentSpendInfo struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_spend_info"`
	// Estimated spend in cents for the current billing month.
	EstimatedSpendCents int64 `json:"estimated_spend_cents"`
	// Monthly spending cap in cents.
	//
	// Null means no cap.
	CapCents *int64 `json:"cap_cents"`
}

// Result of processing a webhook.
type WebhookResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=webhook_response"`
	// Whether the webhook was received and processed.
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

var SampleAgentTokenDetail = &AgentTokenDetail{
	Object:                      constants.ObjectTypeAgentTokenDetail,
	IncludedTokens:              1000000,
	UsedTokens:                  350000,
	InputTokens:                 200000,
	OutputTokens:                150000,
	AdditionalTokensPurchased:   0,
	TotalAvailable:              1000000,
	CurrentPeriodCost:           1.85,
	BillingPeriodEnd:            "2026-04-01T00:00:00Z",
	OverageCostPerMillionTokens: 15.0,
}

var SampleAccountUsageResponse = &AccountUsageResponse{
	Object:           constants.ObjectTypeAccountUsageResponse,
	Seats:            *SampleUsageItem,
	Invoices:         *SampleUsageItemUnlimited,
	Batches:          *SampleUsageItemUnlimited,
	Sandboxes:        *SampleUsageItem,
	Subscription:     SampleSubscriptionInfo,
	AgentTokenDetail: SampleAgentTokenDetail,
}

var SampleBillingPortalSessionResponse = &BillingPortalSessionResponse{
	Object: constants.ObjectTypeBillingPortalSessionResponse,
	URL:    SampleBillingPortalURL,
}

var SampleSwitchPlanResponse = &SwitchPlanResponse{
	Object:  constants.ObjectTypeSwitchPlanResponse,
	Success: true,
}

var SampleEnsureBillingCustomerResponse = &EnsureBillingCustomerResponse{
	Object:           constants.ObjectTypeEnsureBillingCustomerResponse,
	StripeCustomerID: SampleStripeCustomerID,
	Created:          true,
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

func (*AgentTokenDetail) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAgentTokenDetail)
}

func (*SpendingCapResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSpendingCapResponse)
}
