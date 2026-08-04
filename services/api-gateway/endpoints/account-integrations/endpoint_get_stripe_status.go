package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to check Stripe integration status.
type GetStripeStatusRequest struct{}

// Reports whether the target account has a Stripe integration configured, so a checkout flow can tell up front whether card payments are available.
//
// The account is reported as connected whenever Stripe credentials are on file, even if the integration has been deactivated, and the stored keys are not verified against Stripe.
type GetStripeStatusEndpoint struct{}

func (e *GetStripeStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetStripeStatusRequest, *apiresource.StripeStatus] {
	return (&apiendpoint.APIEndpoint[*GetStripeStatusRequest, *apiresource.StripeStatus]{
		Title:             "Get Stripe Status",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/integrations/stripe/status",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetStripeStatusRequest) (*apiresource.StripeStatus, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).GetStripeStatus
		},
	})
}
