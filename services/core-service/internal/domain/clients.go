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
	// CreateTransactionInstantLabel buys carrier labels for a shipment's cases and returns the
	// master tracking number, negotiated rate, and per-case tracking/label details.
	CreateTransactionInstantLabel(ctx context.Context, params CreateLabelParams) (*LabelResult, *apierror.APIError)
	// RefundTransaction refunds a purchased Shippo label transaction (best-effort; used on void).
	RefundTransaction(ctx context.Context, transactionID string) *apierror.APIError
}

// Builds ShippoClient instances from API keys.
type ShippoClientFactory interface {
	Build(apiKey string) ShippoClient
}

// HubspotCompany is the subset of a HubSpot company the sync reads or writes.
type HubspotCompany struct {
	ID     string
	Name   string
	Domain string
	// Lifecycle, when non-empty, sets the company's lifecyclestage (e.g. "customer"). Empty leaves it unchanged.
	Lifecycle string
}

// HubspotContact is the subset of a HubSpot contact the sync reads or writes.
type HubspotContact struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Phone     string
	// Lifecycle, when non-empty, sets the contact's lifecyclestage. Empty leaves it unchanged.
	Lifecycle string
}

// HubspotDeal is the subset of a HubSpot deal the sync reads or writes.
type HubspotDeal struct {
	ID   string
	Name string
	// Amount is the deal value as a decimal string, written to HubSpot's standard `amount` property.
	Amount    string
	CloseDate time.Time
	// PipelineID and StageID select the deal's pipeline and stage (e.g. Closed Won).
	PipelineID string
	StageID    string
	// SalesOrderID is the Augno order id, stored on the deal's augno_sales_order_id property for idempotent upserts.
	SalesOrderID string
}

// HubspotClient performs HubSpot CRM operations for a single account's connected integration.
type HubspotClient interface {
	// EnsureDealProperties creates any custom deal properties the sync depends on (e.g. augno_sales_order_id) if absent. Idempotent.
	EnsureDealProperties(ctx context.Context) *apierror.APIError

	SearchCompaniesByDomain(ctx context.Context, domain string) ([]HubspotCompany, *apierror.APIError)
	SearchCompaniesByName(ctx context.Context, name string) ([]HubspotCompany, *apierror.APIError)
	// ListCompanies returns one page of companies and the cursor for the next page ("" when exhausted). Used by the backfill.
	ListCompanies(ctx context.Context, cursor string) (page []HubspotCompany, next string, err *apierror.APIError)
	CreateCompany(ctx context.Context, company HubspotCompany) (*HubspotCompany, *apierror.APIError)
	UpdateCompany(ctx context.Context, id string, company HubspotCompany) *apierror.APIError

	// UpsertContactByEmail creates or updates a contact keyed on email (HubSpot's native dedupe key).
	UpsertContactByEmail(ctx context.Context, contact HubspotContact) (*HubspotContact, *apierror.APIError)

	// SearchDealBySalesOrderID finds an existing deal by its augno_sales_order_id property, or returns (nil, nil).
	SearchDealBySalesOrderID(ctx context.Context, salesOrderID string) (*HubspotDeal, *apierror.APIError)
	CreateDeal(ctx context.Context, deal HubspotDeal) (*HubspotDeal, *apierror.APIError)
	UpdateDeal(ctx context.Context, id string, deal HubspotDeal) *apierror.APIError

	// Associate links two CRM objects using the default association type (e.g. deals→companies). Types are HubSpot plural object names.
	Associate(ctx context.Context, fromType, fromID, toType, toID string) *apierror.APIError
}

// HubspotClientFactory builds HubspotClient instances from a decrypted access token.
type HubspotClientFactory interface {
	Build(accessToken string) HubspotClient
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
	// UpdateStripeCustomer pushes changed customer details onto an existing Stripe customer. Nil fields are left untouched.
	UpdateStripeCustomer(ctx context.Context, params UpdateStripeCustomerParams) *apierror.APIError
	ConstructWebhookEvent(payload []byte, signature, webhookSecret string) (*StripeWebhookEvent, *StripePaymentIntent, *apierror.APIError)
	// ListPayoutPaymentIntentIDs resolves the payment intent IDs whose charges fund the given payout, by walking the payout's balance transactions (called on payout.paid to stamp funds_received_at).
	ListPayoutPaymentIntentIDs(ctx context.Context, payoutID string) ([]string, *apierror.APIError)
}

// CreateCheckoutSessionParams holds the parameters for creating a Stripe checkout session.
type CreateCheckoutSessionParams struct {
	// StripeCustomerID, when set, bills the session to that existing Stripe customer
	// (and enables saving the payment method). When empty, CustomerEmail is used
	// instead. Stripe rejects supplying both, so exactly one is sent.
	StripeCustomerID string
	CustomerEmail    string
	LineItems        []CheckoutLineItem
	SuccessURL       *string
	CancelURL        *string
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

// PortalDomainProviderState is the serving provider's view of a portal domain: whether it is verified and routing, whether it is actually serving over HTTPS, and which DNS records the customer must publish.
type PortalDomainProviderState struct {
	Verified      bool
	Misconfigured bool
	// Serving reports whether the domain answers over HTTPS with a valid TLS certificate. It is only meaningful once the domain is verified and routing (not misconfigured); until the certificate is issued a routed domain is verified+routing but not yet serving.
	Serving    bool
	DNSRecords []PortalDNSRecord
}

// PortalDomainProvider is the serving/TLS provider (Vercel) for customer portal custom domains. All methods are idempotent so the provider-registration phase of a create can be safely retried.
type PortalDomainProvider interface {
	// AddDomain attaches the domain to the portal project and returns its current state. Adding an already-attached domain succeeds.
	AddDomain(ctx context.Context, domain string) (*PortalDomainProviderState, *apierror.APIError)
	// GetDomainState returns the domain's verification/configuration state and currently required DNS records.
	GetDomainState(ctx context.Context, domain string) (*PortalDomainProviderState, *apierror.APIError)
	// RemoveDomain detaches the domain from the portal project. Removing an unknown domain succeeds.
	RemoveDomain(ctx context.Context, domain string) *apierror.APIError
}
