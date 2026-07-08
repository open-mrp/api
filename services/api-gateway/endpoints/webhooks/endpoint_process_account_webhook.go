package webhooksep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
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

// Processes a Stripe webhook event from an account's connected Stripe account, verifying the signature against the account's stored webhook secret before recording order payments.
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
			SkipRequestLogging:     true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AccountStripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError) {
			return svc.(WebhookSvc).ProcessAccountWebhook
		},
	})
}
