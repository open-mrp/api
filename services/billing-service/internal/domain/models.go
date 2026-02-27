package domain

import "time"

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
	Status            string
	CurrentPeriodEnd  *time.Time
	TrialEnd          *time.Time
	CancelAtPeriodEnd bool
	CancelAt          *time.Time
}

type AccountUsage struct {
	Seats        UsageItem
	Invoices     UsageItem
	Batches      UsageItem
	Sandboxes    UsageItem
	Subscription *SubscriptionInfoResult
}

type ProrationPreview struct {
	CreditAmount                int64
	ChargeAmount                int64
	NetAmount                   int64
	FormattedNetAmount          string
	IsCredit                    bool
	TotalInvoiceAmount          int64
	FormattedTotalInvoiceAmount string
	MonthlyBillAmount           int64
	FormattedMonthlyBillAmount  string
	LineItems                   []ProrationLineItem
}

type ProrationLineItem struct {
	Description string
	Amount      int64
	IsProration bool
}

type RequestEnterpriseUpgradeResult struct {
	Success bool
}

type EnsureBillingCustomerResult struct {
	StripeCustomerID string
	Created          bool
}

type SwitchPlanResult struct {
	Success         bool
	RequiresPayment bool
	CheckoutURL     *string
}

type ConfirmPlanSwitchResult struct {
	Success bool
}

type StripeHostedCheckoutInput struct {
	CustomerID string
	PriceID    string
	Quantity   int64
	SuccessURL string
	CancelURL  string
}

type StripeHostedCheckoutSession struct {
	ID  string
	URL string
}

type AccountSubscriptionInfo struct {
	SubscriptionStatus           *string
	SubscriptionCurrentPeriodEnd *time.Time
	StripeSubscriptionID         *string
}
