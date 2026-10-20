package domain

import (
	"context"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
)

type ListPricingPlansInput struct {
	Cursor *string
	Limit  int32
	Query  *string
}

type ListPricingPlansResult struct {
	Plans    []PricingPlan
	PageInfo pagination.PageInfo
}

type BillingSvc interface {
	// ListPricingPlans returns a paginated list of currently active pricing plans with their limits and features.
	ListPricingPlans(ctx context.Context, input ListPricingPlansInput) (*ListPricingPlansResult, *apierror.APIError)

	// GetPlanByCode returns a single pricing plan by its plan type code.
	GetPlanByCode(ctx context.Context, planCode string) (*PricingPlan, *apierror.APIError)

	// GetAccountUsage returns current resource usage for the given account with plan limits and subscription information.
	GetAccountUsage(ctx context.Context, accountID string) (*AccountUsage, *apierror.APIError)

	// GetAgentSpendCents returns the marked-up token spend the account has accrued this billing period, as it will be billed in Stripe. This is the same figure surfaced on the dashboard; agent-service uses it to enforce the spending cap consistently.
	GetAgentSpendCents(ctx context.Context, accountID string) (int64, *apierror.APIError)

	// GetAgentTokenRates returns the marked-up per-token rates from the account's plan rate card, so agent-service can price a run's in-flight usage against the cap with the same rates Stripe bills (no per-turn round trip). Empty when the account has no pricing plan or rate card.
	GetAgentTokenRates(ctx context.Context, accountID string) ([]TokenRate, *apierror.APIError)

	// CreateBillingPortalSession creates a Stripe billing portal session for managing subscriptions. Returns the portal URL.
	CreateBillingPortalSession(ctx context.Context, accountID string) (string, *apierror.APIError)

	// PreviewPlanChange previews the cost impact of switching to a different pricing plan using the billing intent reserve+void pattern.
	PreviewPlanChange(ctx context.Context, accountID string, planID string) (*PlanChangePreview, *apierror.APIError)

	// RequestEnterpriseUpgrade sends an enterprise plan inquiry to support on behalf of the requesting admin.
	RequestEnterpriseUpgrade(ctx context.Context, input RequestEnterpriseUpgradeInput) (*RequestEnterpriseUpgradeResult, *apierror.APIError)

	// EnsureBillingCustomer links or fetches a Stripe customer for an account. If one already exists it is returned; otherwise a new one is created. Also creates a billing profile if one doesn't exist.
	EnsureBillingCustomer(ctx context.Context, accountID string) (*EnsureBillingCustomerResult, *apierror.APIError)

	// SwitchPlan initiates a plan switch using v2 billing intents.
	SwitchPlan(ctx context.Context, accountID string, planID string) (*SwitchPlanResult, *apierror.APIError)
}

// CoreClient is the interface for calling core-service RPCs from the billing service layer.
type CoreClient interface {
	GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (accountID string, planCode string, apiErr *apierror.APIError)
	UpdateAccountSubscription(ctx context.Context, idempotencyKey, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError
	ClearAccountStripeCustomer(ctx context.Context, idempotencyKey, accountID string) *apierror.APIError
}

type RequestEnterpriseUpgradeInput struct {
	AccountID string
	ActorID   string
	ActorName string
}

type StripeWebhookSvc interface {
	// ProcessWebhookEvent verifies a Stripe webhook signature and enqueues the event for asynchronous processing via the message outbox.
	ProcessWebhookEvent(ctx context.Context, input ProcessWebhookEventInput) (*ProcessWebhookEventResult, *apierror.APIError)
}
