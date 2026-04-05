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
	Query  *string
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

	// PreviewPlanChange previews the cost impact of switching to a different
	// pricing plan using the billing intent reserve+void pattern.
	PreviewPlanChange(ctx context.Context, accountID string, planID string) (*PlanChangePreview, *apierror.APIError)

	// RequestEnterpriseUpgrade sends an enterprise plan inquiry to the sales
	// team on behalf of the requesting admin.
	RequestEnterpriseUpgrade(ctx context.Context, input RequestEnterpriseUpgradeInput) (*RequestEnterpriseUpgradeResult, *apierror.APIError)

	// EnsureBillingCustomer links or fetches a Stripe customer for an account.
	// If one already exists it is returned; otherwise a new one is created.
	// Also creates a billing profile if one doesn't exist.
	EnsureBillingCustomer(ctx context.Context, accountID string) (*EnsureBillingCustomerResult, *apierror.APIError)

	// CreateRegistrationCustomer creates a Stripe customer for a registration
	// session before an account exists. Uses the provided email/name directly.
	CreateRegistrationCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*EnsureBillingCustomerResult, *apierror.APIError)

	// SwitchPlan initiates a plan switch using v2 billing intents.
	SwitchPlan(ctx context.Context, accountID string, planID string) (*SwitchPlanResult, *apierror.APIError)

	// SetupBillingProfile creates a billing profile and cadence for an account.
	SetupBillingProfile(ctx context.Context, accountID string) (*BillingProfileResult, *apierror.APIError)

	// SubscribeToPricingPlan subscribes a Stripe customer to a v2 pricing plan.
	SubscribeToPricingPlan(ctx context.Context, stripeCustomerID, planCode string) *apierror.APIError

	// CreateSetupIntent creates a Stripe Setup Intent for collecting a payment method.
	CreateSetupIntent(ctx context.Context, customerID, idempotencyKey string) (*SetupIntentResult, *apierror.APIError)

	// GetSetupIntentStatus returns the current status of a Stripe Setup Intent.
	GetSetupIntentStatus(ctx context.Context, setupIntentID string) (*SetupIntentResult, *apierror.APIError)

	// ValidateStripePricingPlan checks whether the Stripe pricing plan for a
	// given plan code is accessible. Returns nil if valid or free plan.
	ValidateStripePricingPlan(ctx context.Context, planCode string) *apierror.APIError
}

// SetupIntentResult holds the result of a Setup Intent operation.
type SetupIntentResult struct {
	SetupIntentID   string
	ClientSecret    string // #nosec G117 -- Stripe ephemeral client secret
	Status          string
	PaymentMethodID *string
}

// CoreClient is the interface for calling core-service RPCs from the billing
// service layer.
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
	// ProcessWebhookEvent verifies a Stripe webhook signature and enqueues the
	// event for asynchronous processing via the message outbox.
	ProcessWebhookEvent(ctx context.Context, input ProcessWebhookEventInput) (*ProcessWebhookEventResult, *apierror.APIError)
}
