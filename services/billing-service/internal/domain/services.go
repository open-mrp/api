package domain

import (
	"context"
	"time"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
)

type ListPricingPlansInput struct {
	Cursor *string
	Limit  int32
}

type ListPricingPlansResult struct {
	Plans    []PricingPlan
	PageInfo pagination.PageInfo
}

type BillingSvc interface {
	// ListPricingPlans returns a paginated list of currently active pricing plans
	// with their limits and features.
	ListPricingPlans(ctx context.Context, input ListPricingPlansInput) (*ListPricingPlansResult, *apierror.APIError)

	// GetPlanByCode returns a single pricing plan by its plan type code.
	GetPlanByCode(ctx context.Context, planCode string) (*PricingPlan, *apierror.APIError)

	// GetAccountUsage returns current resource usage for the given account with
	// plan limits and subscription information.
	GetAccountUsage(ctx context.Context, accountID string) (*AccountUsage, *apierror.APIError)

	// CreateBillingPortalSession creates a Stripe billing portal session for
	// managing subscriptions. Returns the portal URL.
	CreateBillingPortalSession(ctx context.Context, accountID string) (string, *apierror.APIError)

	// GetProrationPreview previews the cost impact of switching to a different
	// pricing plan, including proration credits and charges.
	GetProrationPreview(ctx context.Context, accountID string, planID string) (*ProrationPreview, *apierror.APIError)

	// RequestEnterpriseUpgrade sends an enterprise plan inquiry to the sales
	// team on behalf of the requesting admin.
	RequestEnterpriseUpgrade(ctx context.Context, input RequestEnterpriseUpgradeInput) (*RequestEnterpriseUpgradeResult, *apierror.APIError)

	// EnsureBillingCustomer links or fetches a Stripe customer for an account.
	// If one already exists it is returned; otherwise a new one is created.
	EnsureBillingCustomer(ctx context.Context, accountID string) (*EnsureBillingCustomerResult, *apierror.APIError)

	// SwitchPlan initiates a plan switch. Handles free→paid (checkout),
	// paid→free (cancel), and paid→paid (subscription update).
	SwitchPlan(ctx context.Context, accountID string, planID string) (*SwitchPlanResult, *apierror.APIError)

	// ConfirmPlanSwitch confirms a plan upgrade after Stripe checkout completes.
	ConfirmPlanSwitch(ctx context.Context, accountID string, checkoutSessionID string, planID string) (*ConfirmPlanSwitchResult, *apierror.APIError)
}

// CoreClient is the interface for calling core-service RPCs from the billing
// service layer.
type CoreClient interface {
	UpdateAccountSubscription(ctx context.Context, idempotencyKey, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string) *apierror.APIError
}

type RequestEnterpriseUpgradeInput struct {
	AccountID string
	ActorID   string
	ActorName string
}

type StripeWebhookSvc interface {
	// ProcessWebhookEvent verifies a Stripe webhook signature and enqueues the
	// event for asynchronous processing via the message outbox.
	ProcessWebhookEvent(ctx context.Context, input ProcessWebhookEventInput) (*ProcessWebhookEventResult, *apierror.APIError)
}

// CreateCheckoutSessionInput holds the parameters for creating a Stripe
// checkout session through the billing service.
type CreateCheckoutSessionInput struct {
	CustomerID     string
	PlanCode       string
	ReturnURL      string
	IdempotencyKey string
}

// CreateCheckoutSessionResult holds the result of creating a Stripe checkout
// session, including the publishable key for the frontend.
type CreateCheckoutSessionResult struct {
	SessionID      string
	ClientSecret   string // #nosec G117 - Stripe checkout client secret (ephemeral, not a stored credential)
	PublishableKey string
}

// GetCheckoutSessionStatusInput holds the parameters for retrieving a Stripe
// checkout session's status.
type GetCheckoutSessionStatusInput struct {
	CheckoutSessionID string
}

// GetCheckoutSessionStatusResult holds the status of a Stripe checkout session
// and, when complete, the resulting subscription and customer IDs.
type GetCheckoutSessionStatusResult struct {
	Status         string
	SubscriptionID string
	CustomerID     string
}

// CheckoutSvc handles Stripe checkout operations: creating customers and
// checkout sessions. It encapsulates the product/price lookup and creation
// logic that was previously in the auth service.
type CheckoutSvc interface {
	// CreateCustomer creates a Stripe customer with the given email, name, and
	// metadata. Returns the Stripe customer ID.
	CreateCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*StripeCustomer, *apierror.APIError)

	// CreateCheckoutSession looks up the plan by code, ensures the corresponding
	// Stripe product and price exist, and creates an embedded checkout session.
	// Returns the session details along with the Stripe publishable key.
	CreateCheckoutSession(ctx context.Context, input CreateCheckoutSessionInput) (*CreateCheckoutSessionResult, *apierror.APIError)

	// GetCheckoutSessionStatus retrieves the current status of a Stripe checkout
	// session. Returns the status along with subscription and customer IDs when
	// the session is complete.
	GetCheckoutSessionStatus(ctx context.Context, input GetCheckoutSessionStatusInput) (*GetCheckoutSessionStatusResult, *apierror.APIError)
}
