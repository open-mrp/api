package domain

import (
	"context"
	"fmt"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

// StripeClient is the interface for all Stripe API operations needed by the billing service, including v2 pricing plan billing and customer management.
type StripeClient interface {
	// V1 APIs (still needed)
	VerifyWebhookSignature(payload []byte, signature string) (*StripeEvent, error)
	CreateCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*StripeCustomer, error)
	CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (*StripeBillingPortalSession, error)

	// V2 Pricing Plan APIs
	GetPricingPlan(ctx context.Context, pricingPlanID string) (*StripePricingPlan, error)
	CreateBillingProfile(ctx context.Context, customerID, idempotencyKey string) (profileID string, err error)
	CreateBillingCadence(ctx context.Context, billingProfileID, idempotencyKey string) (cadenceID string, err error)
	CreateBillingIntent(ctx context.Context, cadenceID string, actions []BillingIntentAction, idempotencyKey string) (intentID string, err error)
	ReserveBillingIntent(ctx context.Context, intentID string) (*BillingIntentReservation, error)
	CreatePaymentIntent(ctx context.Context, amountCents int64, currency, customerID, returnURL string) (paymentIntentID string, err error)
	CommitBillingIntent(ctx context.Context, intentID string, paymentIntentID *string, cadenceID string) (*BillingIntentCommitResult, error)
	VoidBillingIntent(ctx context.Context, intentID string) error

	// FetchObject fetches a Stripe object by its API path (used for v2 thin event related_object).
	FetchObject(ctx context.Context, objectURL string) ([]byte, error)

	// CreateSetupIntent creates a Stripe Setup Intent for collecting a payment method.
	CreateSetupIntent(ctx context.Context, customerID, idempotencyKey string) (*StripeSetupIntent, error)
	// GetSetupIntent retrieves the current state of a Stripe Setup Intent.
	GetSetupIntent(ctx context.Context, setupIntentID string) (*StripeSetupIntent, error)

	// ReportMeterEvent reports a usage meter event to the Stripe V2 billing/meter_events API.
	ReportMeterEvent(ctx context.Context, eventName, stripeCustomerID string, value int, idempotencyKey string) error

	// GetAgentTokenSpendCents returns the marked-up cost in cents of the customer's metered LLM token usage since the given time, reconstructed from the plan's rate card rates and the Stripe usage meter. This equals what Stripe will bill for the metered token lines: rates carry the plan's markup and the AI Gateway meters real per-model, per-token-type usage (including cache). Returns 0 when the rate card has no rates or the customer has no usage in the window.
	GetAgentTokenSpendCents(ctx context.Context, customerID, rateCardID string, since time.Time) (int64, error)

	// GetRateCardTokenRates returns every marked-up per-token rate on a rate card, keyed by (model, token_type). These are the same rates GetAgentTokenSpendCents prices usage against; callers use them to price in-flight usage without a Stripe round trip.
	GetRateCardTokenRates(ctx context.Context, rateCardID string) ([]TokenRate, error)

	// CreateCreditGrant grants prepaid billing credits (amountCents, USD) to a customer for a purchased token pack. The grant is scoped to metered usage and never expires, so it draws down against the plan's LLM-token rate card as agents run. idempotencyKey guards duplicate grants on webhook redelivery. Returns the Stripe credit grant id.
	CreateCreditGrant(ctx context.Context, customerID string, amountCents int64, name, idempotencyKey string) (grantID string, err error)

	// GetCreditGrantBalanceCents returns the customer's available prepaid credit balance in cents, summed across metered-scoped credit grants. Drives the dashboard balance display and the agent runner's prepaid gate; the burndown itself happens inside Stripe.
	GetCreditGrantBalanceCents(ctx context.Context, customerID string) (int64, error)
}

// TokenRate is a marked-up per-token price from a plan's rate card.
type TokenRate struct {
	// Model is the gateway model name the rate applies to (e.g. "anthropic/claude-sonnet-4.6").
	Model string
	// TokenType is the token type the rate applies to: input, output, cached_input, or cached_output.
	TokenType string
	// UnitAmountCents is the price in cents per token, markup included.
	UnitAmountCents float64
}

// StripeCustomer represents a Stripe customer created during registration.
type StripeCustomer struct {
	ID string
}

type StripeEvent struct {
	ID       string
	Type     string
	ObjectID string
	Data     []byte
}

// StripeBillingPortalSession represents a created Stripe billing portal session.
type StripeBillingPortalSession struct {
	URL string
}

// StripePricingPlan represents a v2 pricing plan with its live version.
type StripePricingPlan struct {
	ID                    string
	LiveVersion           string
	LicenseFeeComponentID string
	// RateCardID is the id (rcd_...) of the rate card component attached to the plan's live version, empty when the plan has no rate card. It prices the plan's metered LLM token usage (markup baked in).
	RateCardID string
	// DisplayName is the plan's human-facing name in Stripe (e.g. "Founder").
	DisplayName string
	// BaseFeeCents is the flat recurring fee in cents from the plan's license fee component, 0 when the plan has none. BaseFeeInterval is the interval it recurs on (e.g. "month").
	BaseFeeCents    int64
	BaseFeeInterval string
}

// BillingIntentAction represents an action in a billing intent (subscribe, modify, deactivate).
type BillingIntentAction struct {
	Type                    string // "subscribe", "modify", "deactivate"
	PricingPlanID           string
	PricingPlanVersion      string
	SubscriptionID          string // for modify/deactivate
	ComponentConfigurations []ComponentConfiguration
}

// ComponentConfiguration sets the quantity for a pricing plan component (e.g. license fee seats).
type ComponentConfiguration struct {
	PricingPlanComponentID string
	Quantity               int
}

// BillingIntentReservation holds the result of reserving a billing intent.
type BillingIntentReservation struct {
	IntentID  string
	NetAmount int64
	LineItems []BillingIntentLineItem
}

// BillingIntentLineItem represents a line item from a billing intent reservation.
type BillingIntentLineItem struct {
	Description string
	Amount      int64
}

// ErrBillingIntentConflict is returned when CreateBillingIntent fails because a pricing plan subscription is already reserved by another billing intent.
type ErrBillingIntentConflict struct {
	ConflictingIntentID string
	Err                 error
}

func (e *ErrBillingIntentConflict) Error() string {
	return fmt.Sprintf("billing intent conflict: subscription reserved by %s", e.ConflictingIntentID)
}

func (e *ErrBillingIntentConflict) Unwrap() error {
	return e.Err
}

// BillingIntentCommitResult holds IDs extracted from a committed billing intent.
type BillingIntentCommitResult struct {
	PricingPlanSubscriptionIDs []string
}

// StripeSetupIntent represents a Stripe Setup Intent for collecting payment methods.
type StripeSetupIntent struct {
	ID              string
	ClientSecret    string // #nosec G117 -- Stripe ephemeral client secret
	Status          string
	PaymentMethodID *string
}

// NotificationClient is the interface for calling notification-service RPCs.
type NotificationClient interface {
	SendEnterpriseRequest(ctx context.Context, accountID, accountName, currentPlanName, requesterName, requesterEmail string) *apierror.APIError
	SendPaymentActionRequired(ctx context.Context, accountID, adminEmail string) *apierror.APIError
}
