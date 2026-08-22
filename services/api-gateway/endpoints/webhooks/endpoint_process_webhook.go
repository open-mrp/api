package webhooksep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Processes a Stripe webhook event delivered to OpenMRP's own Stripe account.
//
// The payload's signature is verified, then the events OpenMRP acts on — subscription servicing and collection changes, billing cadence outcomes, customer deletions, and completed checkouts — are queued for asynchronous handling; every other event type is acknowledged and dropped. A success response means the event was accepted, not that it has already been applied to the account.
type ProcessWebhookEndpoint struct{}

func (e *ProcessWebhookEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.StripeWebhookRequest, *apiresource.WebhookResponse] {
	return (&apiendpoint.APIEndpoint[*apiresource.StripeWebhookRequest, *apiresource.WebhookResponse]{
		Title:             "Process Stripe Webhook",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/webhooks/stripe",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
			HideFromRequestLog:     true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.StripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError) {
			return svc.(WebhookSvc).ProcessWebhook
		},
	})
}
