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
	// Account ID from the URL path.
	AccountID string `path:"accountID" validate:"required"`
}

type ProcessAccountWebhookEndpoint struct{}

func (e *ProcessAccountWebhookEndpoint) Materialize() *apiendpoint.APIEndpoint[*AccountStripeWebhookRequest, *apiresource.WebhookResponse] {
	return &apiendpoint.APIEndpoint[*AccountStripeWebhookRequest, *apiresource.WebhookResponse]{
		Title:             "Process Account Stripe Webhook",
		Description:       "Processes a Stripe webhook event for an account, verifying the signature using the account's credentials.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
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
