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

// Returns the Stripe publishable key for the target account.
type GetStripePublishableKeyEndpoint struct{}

func (e *GetStripePublishableKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetStripePublishableKeyRequest, *apiresource.StripePublishableKey] {
	return (&apiendpoint.APIEndpoint[*GetStripePublishableKeyRequest, *apiresource.StripePublishableKey]{
		Title:             "Get Stripe Publishable Key",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/integrations/stripe/publishable-key",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetStripePublishableKeyRequest) (*apiresource.StripePublishableKey, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).GetStripePublishableKey
		},
	})
}
