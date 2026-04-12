package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetStripePublishableKeyRequest is the request to retrieve the Stripe publishable key.
type GetStripePublishableKeyRequest struct{}

type GetStripePublishableKeyEndpoint struct{}

func (e *GetStripePublishableKeyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetStripePublishableKeyRequest, *apiresource.StripePublishableKey] {
	return &apiendpoint.APIEndpoint[*GetStripePublishableKeyRequest, *apiresource.StripePublishableKey]{
		Title:             "Get Stripe Publishable Key",
		Description:       "Returns the Stripe publishable key for the target account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/integrations/stripe/publishable-key",
		Request:           &GetStripePublishableKeyRequest{},
		Response:          &apiresource.StripePublishableKey{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetStripePublishableKeyRequest) (*apiresource.StripePublishableKey, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).GetStripePublishableKey
		},
	}
}
