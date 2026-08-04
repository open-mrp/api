package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve the Stripe publishable key.
type GetStripePublishableKeyRequest struct{}

// Returns the Stripe publishable key for the target account, for initializing Stripe in a client-side checkout.
//
// Only the publishable key is exposed; the account's secret key and webhook secret never leave the platform. Fails if the account has no Stripe integration or the Stripe integration is inactive.
type GetStripePublishableKeyEndpoint struct{}

func (e *GetStripePublishableKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetStripePublishableKeyRequest, *apiresource.StripePublishableKey] {
	return (&apiendpoint.APIEndpoint[*GetStripePublishableKeyRequest, *apiresource.StripePublishableKey]{
		Title:             "Get Stripe Publishable Key",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/integrations/stripe/publishable-key",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetStripePublishableKeyRequest) (*apiresource.StripePublishableKey, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).GetStripePublishableKey
		},
	})
}
