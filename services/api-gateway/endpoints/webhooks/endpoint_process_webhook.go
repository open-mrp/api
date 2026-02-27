package webhooksep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const processWebhookDescription string = `Receives and processes Stripe webhook events. This endpoint verifies the webhook signature and dispatches events for processing.`

type ProcessWebhookEndpoint struct{}

func (e *ProcessWebhookEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.StripeWebhookRequest, *apiresource.WebhookResponse] {
	return &apiendpoint.APIEndpoint[*apiresource.StripeWebhookRequest, *apiresource.WebhookResponse]{
		Title:             "Process Stripe Webhook",
		Description:       processWebhookDescription,
		Method:            http.MethodPost,
		Route:             "/v1/webhooks/stripe",
		Request:           &apiresource.StripeWebhookRequest{},
		Response:          &apiresource.WebhookResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestBodyParsing: true,
			SkipRequestLogging:     true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.StripeWebhookRequest) (*apiresource.WebhookResponse, *apierror.APIError) {
			return svc.(WebhookSvc).ProcessWebhook
		},
	}
}
