package domain

import (
	"context"
	"fmt"

	apierror "github.com/augno/api/shared/errors"
)

// StripeClient is the interface for all Stripe API operations needed by the
// billing service, including v2 pricing plan billing and customer management.
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

// ErrBillingIntentConflict is returned when CreateBillingIntent fails because
// a pricing plan subscription is already reserved by another billing intent.
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
