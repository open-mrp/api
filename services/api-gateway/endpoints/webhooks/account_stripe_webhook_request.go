package webhooksep

// Request for per-account Stripe webhook processing.
type AccountStripeWebhookRequest struct {
	// Raw request body bytes for signature verification.
	RawBody []byte `rawbody:"true"`
	// Stripe-Signature header value for payload verification.
	Signature string `header:"Stripe-Signature"`
	// Account ID from the URL path.
	AccountID string `path:"account_id" validate:"required"`
}
