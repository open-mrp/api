package domain

// StripeEventLog represents a deduplicated record of a processed Stripe event.
type StripeEventLog struct {
	ID        string
	EventID   string
	ObjectID  string
	EventType string
}

// OrderPaymentIntent links a Stripe payment intent to a sales order.
type OrderPaymentIntent struct {
	ID              string
	PaymentIntentID string
	SalesOrderID    string
}

// TransactionRecord represents a payment transaction.
type TransactionRecord struct {
	ID       string
	Number   string
	AmountID string
}

// CreateCustomerCheckoutSessionParams holds the parameters for customer checkout.
type CreateCustomerCheckoutSessionParams struct {
	OrderID         string
	OrderNumber     string
	OrderTotalCents int64
	CustomerPO      *string
}

// CreateCustomerCheckoutSessionResult holds the result of customer checkout.
type CreateCustomerCheckoutSessionResult struct {
	ClientSecret string // #nosec G117 -- Stripe ephemeral client secret
}

// HandleStripeWebhookParams holds the parameters for processing an account Stripe webhook.
type HandleStripeWebhookParams struct {
	AccountID       string
	RawPayload      []byte
	StripeSignature string
}

// StripeEmbeddedCheckoutSession represents an embedded Stripe checkout session result.
type StripeEmbeddedCheckoutSession struct {
	ClientSecret string // #nosec G117 -- Stripe ephemeral client secret
}

// CreateEmbeddedCheckoutSessionParams holds the parameters for creating an embedded checkout.
type CreateEmbeddedCheckoutSessionParams struct {
	StripeCustomerID string
	AccountSlug      string
	CustomerID       string
	OrderNumber      string
	CustomerPO       *string
	OrderTotalCents  int64
	OrderID          string
	ReturnURL        string
}

// CreateStripeCustomerParams holds the parameters for creating a Stripe customer.
type CreateStripeCustomerParams struct {
	Email      string
	Name       string
	Number     string
	CustomerID string
}

// StripeCustomer represents a Stripe customer.
type StripeCustomer struct {
	ID string
}

// UpdateStripeCustomerParams holds the parameters for updating a Stripe customer.
type UpdateStripeCustomerParams struct {
	StripeCustomerID string
	Email            *string
	Name             *string
	Number           *string
}

// StripeWebhookEvent represents a parsed Stripe webhook event.
type StripeWebhookEvent struct {
	ID      string
	Type    string
	RawJSON []byte
}

// StripePaymentIntent represents a Stripe payment intent extracted from a webhook event.
type StripePaymentIntent struct {
	ID                 string
	Amount             int64
	PaymentMethodTypes []string
	Metadata           map[string]string
}

// TransactionMethodCode maps Stripe payment method types to internal codes.
type TransactionMethodCode string

const (
	TransactionMethodCreditCard TransactionMethodCode = "credit_card"
	TransactionMethodACH        TransactionMethodCode = "ach"
)
