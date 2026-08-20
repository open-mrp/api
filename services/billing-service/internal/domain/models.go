package domain

import (
	"encoding/json"
	"time"
)

type PricingPlan struct {
	ID                   int64
	CreatedAt            time.Time
	TypeID               string
	Name                 string
	PlanTypeCode         string
	PricePerSeat         float64
	PricePerMonth        *float64
	SeatMinimum          *int
	Limits               []PlanLimit
	DisplayFeatures      []string
	DisplayOrder         int
	IsHighlighted        bool
	ButtonText           string
	IncludesPreviousPlan *string
	StripePricingPlanID  *string
}

type PlanLimit struct {
	Key   string
	Value *int
}

type ProcessWebhookEventInput struct {
	RawPayload      []byte
	StripeSignature string
}

type ProcessWebhookEventResult struct {
	Success bool
}

type UsageItem struct {
	Current int
	Limit   *int
}

type SubscriptionInfoResult struct {
	ServicingStatus  string
	CollectionStatus string
}

type AccountUsage struct {
	Seats                    UsageItem
	Invoices                 UsageItem
	Batches                  UsageItem
	Sandboxes                UsageItem
	Subscription             *SubscriptionInfoResult
	EstimatedAgentSpendCents int64
	// PlanName is the pricing plan's display name resolved live from Stripe; empty when the account has no Stripe pricing plan.
	PlanName string
	// BaseFeeCents is the flat base fee charged per BaseFeeInterval, resolved from the plan's license fee component; 0 when the plan has no base fee.
	BaseFeeCents int64
	// BaseFeeInterval is the interval the base fee is charged on (e.g. "month"); empty when there is no base fee.
	BaseFeeInterval string
}

type PlanChangePreview struct {
	NetAmount                  int64
	FormattedNetAmount         string
	MonthlyBillAmount          int64
	FormattedMonthlyBillAmount string
	LineItems                  []PlanChangePreviewLineItem
	IsEstimate                 bool
}

type PlanChangePreviewLineItem struct {
	Description string
	Amount      int64
}

type RequestEnterpriseUpgradeResult struct {
	Success bool
}

type EnsureBillingCustomerResult struct {
	StripeCustomerID string
	Created          bool
	BillingProfileID *string
}

type SwitchPlanResult struct {
	Success  bool
	IntentID *string
}

type BillingProfileResult struct {
	ProfileID string
	CadenceID string
}

type AccountSubscriptionInfo struct {
	SubscriptionStatus           *string
	SubscriptionCurrentPeriodEnd *time.Time
	StripeSubscriptionID         *string
	ServicingStatus              *string
	CollectionStatus             *string
	BillingProfileID             *string
	BillingCadenceID             *string
	PricingPlanSubscriptionID    *string
}

// Returns the subscription's current period end, or nil when the account has no subscription at all.
func (i *AccountSubscriptionInfo) PeriodEnd() *time.Time {
	if i == nil {
		return nil
	}
	return i.SubscriptionCurrentPeriodEnd
}

type IdempotencyKey struct {
	ID             int64
	TypeID         string
	ServiceName    string
	Handler        string
	IdempotencyKey string
	ActorID        *string
	IdentityType   string
	ScopeHash      string
	ResponseCode   *int
	ResponseBody   json.RawMessage
	RecoveryPoint  string
}

func (k *IdempotencyKey) HasResponse() bool {
	return k.ResponseCode != nil
}

func (k *IdempotencyKey) IsFinished() bool {
	return k.RecoveryPoint == string(RecoveryPointFinished)
}

type AgentTokenBilling struct {
	ID                     string
	AccountID              string
	PeriodStart            time.Time
	PeriodEnd              time.Time
	TotalInputTokens       int64
	TotalOutputTokens      int64
	TotalTokens            int64
	TokensReportedToStripe int64
	StripeMeteredItemID    *string
	RunCount               int
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
