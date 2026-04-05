package webhooksep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AccountStripeWebhookRequest carries the raw body, signature, and account ID for per-account Stripe webhook verification.
type AccountStripeWebhookRequest struct {
	// The raw request body bytes for webhook signature verification.
	RawBody []byte `rawbody:"true"`
	// The Stripe-Signature header value used to verify the webhook payload.
	Signature string `header:"Stripe-Signature"`
	// The account ID from the URL path.
	AccountID string `path:"accountID" validate:"required"`
}

type ProcessAccountWebhookEndpoint struct{}

func (e *ProcessAccountWebhookEndpoint) Materialize() *apiendpoint.APIEndpoint[*AccountStripeWebhookRequest, *apiresource.WebhookResponse] {
	return &apiendpoint.APIEndpoint[*AccountStripeWebhookRequest, *apiresource.WebhookResponse]{
		Title:             "Process Account Stripe Webhook",
		Description:       "Receives and processes a Stripe webhook event for a specific account, verifying the signature using the account's Stripe credentials.",
		Method:            http.MethodPost,
		Route:             "/v1/webhooks/stripe/{accountID}",
		Request:           &AccountStripeWebhookRequest{},
		Response:          &apiresource.WebhookResponse{},
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
	}
}
