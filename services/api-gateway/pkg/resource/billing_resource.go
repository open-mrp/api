package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

const SampleStripeCustomerID = "cus_OG9R5zKr5xJHHp"
const SampleBillingPortalURL = "https://billing.stripe.com/p/session/test_YWNjdF8xTTJKVGtMa3E0Z3Bic"

// UsageItem represents a single usage metric with current value and optional limit.
type UsageItem struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=usage_item"`
	// The current usage count.
	Current int `json:"current"`
	// The maximum allowed usage, null means unlimited.
	Limit *int `json:"limit"`
}

// AgentTokenDetail provides detailed agent token usage information.
type AgentTokenDetail struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_token_detail"`
	// Number of tokens included in the plan.
	IncludedTokens int64 `json:"included_tokens"`
	// Total tokens used in the current period.
	UsedTokens int64 `json:"used_tokens"`
	// Input tokens used in the current period.
	InputTokens int64 `json:"input_tokens"`
	// Output tokens used in the current period.
	OutputTokens int64 `json:"output_tokens"`
	// Additional tokens purchased via token packs.
	AdditionalTokensPurchased int64 `json:"additional_tokens_purchased"`
	// Total tokens available (included + purchased).
	TotalAvailable int64 `json:"total_available"`
	// Estimated cost in dollars for the current period.
	CurrentPeriodCost float64 `json:"current_period_cost"`
	// When the current billing period ends (ISO 8601).
	BillingPeriodEnd string `json:"billing_period_end"`
	// Cost per million tokens for overage usage.
	OverageCostPerMillionTokens float64 `json:"overage_cost_per_million_tokens"`
}

// AccountUsageResponse represents account usage metrics across all resource types.
type AccountUsageResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_usage_response"`
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
	// Estimated agent LLM spending for the current month.
	AgentSpend *AgentSpendInfo `json:"agent_spend"`
	// Detailed agent token usage breakdown.
	AgentTokenDetail *AgentTokenDetail `json:"agent_token_detail"`
}

// SubscriptionInfo represents v2 subscription status information.
type SubscriptionInfo struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=subscription_info"`
	// The servicing status of the pricing plan subscription (e.g., "active", "canceled").
	ServicingStatus string `json:"servicing_status" validate:"required"`
	// The collection status (e.g., "current", "paused").
	CollectionStatus string `json:"collection_status" validate:"required"`
}

// BillingPortalSessionResponse represents a Stripe billing portal session.
type BillingPortalSessionResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=billing_portal_session_response"`
	// The URL to redirect the user to the Stripe billing portal.
	URL string `json:"url" validate:"required"`
}

// SwitchPlanResponse represents the result of initiating a plan switch.
type SwitchPlanResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=switch_plan_response"`
	// Whether the plan switch was initiated successfully.
	Success bool `json:"success"`
	// The billing intent ID, if a v2 billing intent was created.
	IntentID *string `json:"intent_id"`
}

// EnsureBillingCustomerResponse represents the result of ensuring a billing customer exists.
type EnsureBillingCustomerResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=ensure_billing_customer_response"`
	// The ID of the Stripe customer for billing.
	StripeCustomerID string `json:"stripe_customer_id" validate:"required"`
	// Indicates whether a new Stripe customer was created.
	Created bool `json:"created"`
	// The billing profile ID, if one was created.
	BillingProfileID *string `json:"billing_profile_id"`
}

// SpendingCapResponse represents the monthly agent spending cap for an account.
type SpendingCapResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=spending_cap_response"`
	// The monthly spending cap in cents. Null means no cap (unlimited).
	CapCents *int64 `json:"cap_cents"`
}

// AgentSpendInfo provides estimated agent LLM spending for the current month.
type AgentSpendInfo struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=agent_spend_info"`
	// The estimated spend in cents for the current billing month.
	EstimatedSpendCents int64 `json:"estimated_spend_cents"`
	// The monthly spending cap in cents. Null means no cap.
	CapCents *int64 `json:"cap_cents"`
}

// WebhookResponse represents the result of processing a webhook.
type WebhookResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=webhook_response"`
	// Whether the webhook was received and processed.
	Received bool `json:"received"`
}

// StripeWebhookRequest carries the raw body and signature for Stripe webhook verification.
type StripeWebhookRequest struct {
	// The raw request body bytes for webhook signature verification.
	RawBody []byte `rawbody:"true"`
	// The Stripe-Signature header value used to verify the webhook payload.
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
