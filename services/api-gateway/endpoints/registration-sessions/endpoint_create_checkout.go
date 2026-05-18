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

// Creates a Stripe customer and billing profile for a registration session.
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
