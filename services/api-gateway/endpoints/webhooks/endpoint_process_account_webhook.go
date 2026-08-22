package webhooksep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request for per-account Stripe webhook processing.
type AccountStripeWebhookRequest struct {
	// Raw request body bytes for signature verification.
	RawBody []byte `rawbody:"true"`
	// Stripe-Signature header value for payload verification.
	Signature string `header:"Stripe-Signature"`
	// The account these Stripe webhook events belong to.
	AccountID string `path:"account_id" validate:"required"`
}

// Processes a Stripe webhook event delivered by an account's own connected Stripe account.
//
// The payload is verified against the webhook secret stored on that account's Stripe integration. A succeeded payment is linked to the sales order it references and recorded as a customer payment; failed and canceled payments undo that link, and a completed payout marks the payments it covers as having reached the account's bank. Other event types, and payments that reference an order the account does not own, are acknowledged without action. Processing is idempotent, so Stripe's retries are safe.
type ProcessAccountWebhookEndpoint struct{}

func (e *ProcessAccountWebhookEndpoint) Materialize() *apiendpoint.APIEndpoint[*AccountStripeWebhookRequest, *apiresource.WebhookResponse] {
	return (&apiendpoint.APIEndpoint[*AccountStripeWebhookRequest, *apiresource.WebhookResponse]{
		Title:             "Process Account Stripe Webhook",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/webhooks/stripe/accounts/{account_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
			HideFromRequestLog:     true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AccountStripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError) {
			return svc.(WebhookSvc).ProcessAccountWebhook
		},
	})
}
