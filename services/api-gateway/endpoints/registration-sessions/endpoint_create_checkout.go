package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to set up billing for a registration session.
type SetupBillingRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

// Creates a Stripe customer and Setup Intent for collecting the registration's payment method.
//
// Returns the Setup Intent client secret and publishable key needed to collect a payment method with Stripe.js. The Stripe customer is created once and reused on later calls, but every call issues a new Setup Intent and replaces the one recorded on the session, so confirm payment with the Setup Intent from the most recent call. Rejected once the registration has completed.
type SetupBillingEndpoint struct{}

func (e *SetupBillingEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetupBillingRequest, *apiresource.SetupBillingResponse] {
	return (&apiendpoint.APIEndpoint[*SetupBillingRequest, *apiresource.SetupBillingResponse]{
		Title:             "Setup Registration Billing",
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/actions/setup-billing",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetupBillingRequest) (*apiresource.SetupBillingResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).SetupBilling
		},
	})
}
