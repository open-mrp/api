package domain

import (
	"context"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

// IncompleteRegistrationSession is the subset of a pending registration session returned to the tenancy response.
type IncompleteRegistrationSession struct {
	SessionID string
	PlanCode  string
	Step      string
	CreatedAt time.Time
}

// CoreAuthClient is the core-service's client for calling into auth-service.
type CoreAuthClient interface {
	// GetIncompleteRegistrationSession returns the user's most recent incomplete registration session, or (nil, nil) if none exists.
	GetIncompleteRegistrationSession(ctx context.Context, userID string) (*IncompleteRegistrationSession, *apierror.APIError)
}

// ShippoCarrierAccount represents a carrier account registered with Shippo.
type ShippoCarrierAccount struct {
	ObjectID        string
	Carrier         string
	AccountID       string
	Active          bool
	IsShippoAccount bool
}

// ShippoServiceLevel represents a service level returned by Shippo.
type ShippoServiceLevel struct {
	Name  string
	Token string
}

// ShippoClient defines the interface for interacting with the Shippo API.
type ShippoClient interface {
	FindOrRegisterCarrierAccount(ctx context.Context, carrier string) (*ShippoCarrierAccount, *apierror.APIError)
	ConnectCarrierAccount(ctx context.Context, carrier, accountID string, params map[string]string) (*ShippoCarrierAccount, *apierror.APIError)
	GetCarrierAccount(ctx context.Context, objectID string) (*ShippoCarrierAccount, *apierror.APIError)
	DeactivateCarrierAccount(ctx context.Context, objectID string) *apierror.APIError
	GetCarrierServiceLevels(ctx context.Context, objectID string) ([]ShippoServiceLevel, *apierror.APIError)
	InitiateOAuth(ctx context.Context, objectID, redirectURI string, state *string) (string, *apierror.APIError)
	FetchShippingRate(ctx context.Context, params FetchShippingRateParams) (float64, *apierror.APIError)
	FetchAllShippingRates(ctx context.Context, params FetchAllShippingRatesParams) ([]ShippoRateOption, *apierror.APIError)
}

// ShippoClientFactory builds ShippoClient instances from API keys.
type ShippoClientFactory interface {
	Build(apiKey string) ShippoClient
}

// StripeCheckoutSession represents the result of creating a Stripe checkout session.
type StripeCheckoutSession struct {
	URL string
}

// StripeCheckoutClient provides Stripe checkout session creation for per-account Stripe integrations.
type StripeCheckoutClient interface {
	CreateOneTimeCheckoutSession(ctx context.Context, params CreateCheckoutSessionParams) (*StripeCheckoutSession, *apierror.APIError)
	CreateEmbeddedCheckoutSession(ctx context.Context, params CreateEmbeddedCheckoutSessionParams) (*StripeEmbeddedCheckoutSession, *apierror.APIError)
	CreateStripeCustomer(ctx context.Context, params CreateStripeCustomerParams) (*StripeCustomer, *apierror.APIError)
	ConstructWebhookEvent(payload []byte, signature, webhookSecret string) (*StripeWebhookEvent, *StripePaymentIntent, *apierror.APIError)
}

// CreateCheckoutSessionParams holds the parameters for creating a Stripe checkout session.
type CreateCheckoutSessionParams struct {
	CustomerEmail string
	LineItems     []CheckoutLineItem
	SuccessURL    *string
	CancelURL     *string
	// Metadata to attach to the payment intent (e.g. orderID, customerID).
	PaymentIntentMetadata map[string]string
}

// CheckoutLineItem represents a line item in a Stripe checkout session.
type CheckoutLineItem struct {
	Name        string
	Description string
	AmountCents int64
	Quantity    int64
}

// StripeCheckoutClientFactory builds StripeCheckoutClient instances from API keys.
type StripeCheckoutClientFactory interface {
	Build(apiKey string) StripeCheckoutClient
}
