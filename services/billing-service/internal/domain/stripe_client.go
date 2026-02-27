package domain

import (
	"context"
	"time"

	"github.com/stripe/stripe-go/v84"
)

// StripeClient is the interface for all Stripe API operations needed by the
// billing service, including webhook verification, subscription lookups, and
// checkout flows (customer/product/price/session creation).
type StripeClient interface {
	VerifyWebhookSignature(payload []byte, signature string) (*StripeEvent, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*StripeSubscription, error)
	CreateCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*StripeCustomer, error)
	GetOrCreateProduct(ctx context.Context, planCode, planName, idempotencyKey string) (*stripe.Product, error)
	GetOrCreatePrice(ctx context.Context, productID string, unitAmount int64, planCode, idempotencyKey string) (*stripe.Price, error)
	CreateCheckoutSession(ctx context.Context, input StripeCheckoutSessionInput) (*StripeCheckoutSession, error)
	GetCheckoutSession(ctx context.Context, sessionID string) (*StripeCheckoutSessionStatus, error)
	CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (*StripeBillingPortalSession, error)
	ListSubscriptions(ctx context.Context, customerID string) ([]*stripe.Subscription, error)
	GetCustomerBalance(ctx context.Context, customerID string) (int64, error)
	CreateInvoicePreview(ctx context.Context, customerID string, subscriptionID string, items []*stripe.InvoiceCreatePreviewSubscriptionDetailsItemParams) (*stripe.Invoice, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	UpdateSubscription(ctx context.Context, subscriptionID string, items []*stripe.SubscriptionItemsParams) (*stripe.Subscription, error)
	CreateHostedCheckoutSession(ctx context.Context, input StripeHostedCheckoutInput) (*StripeHostedCheckoutSession, error)
	CreateSubscription(ctx context.Context, customerID, priceID string, quantity int64, defaultPaymentMethodID string) (*StripeSubscription, error)
	ListPaymentMethods(ctx context.Context, customerID string) ([]string, error)
}

// StripeCustomer represents a Stripe customer created during registration.
type StripeCustomer struct {
	ID string
}

// StripeCheckoutSessionInput holds the low-level parameters for creating a
// Stripe checkout session via the Stripe API.
type StripeCheckoutSessionInput struct {
	CustomerID     string
	PriceID        string
	Quantity       int64
	TrialDays      int64
	IdempotencyKey string
	ReturnURL      string
}

// StripeCheckoutSession represents a created Stripe checkout session.
type StripeCheckoutSession struct {
	ID           string
	ClientSecret string // #nosec G117 - Stripe checkout client secret (ephemeral, not a stored credential)
}

// StripeCheckoutSessionStatus holds the status of a Stripe checkout session
// after retrieval, including the subscription and customer IDs when complete.
type StripeCheckoutSessionStatus struct {
	Status         string
	SubscriptionID string
	CustomerID     string
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

type StripeSubscription struct {
	ID                string
	CustomerID        string
	Status            string
	PlanCode          string
	CurrentPeriodEnd  time.Time
	TrialEnd          *time.Time
	CancelAtPeriodEnd bool
	CancelAt          *time.Time
}
