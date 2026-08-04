package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to confirm payment for a registration session.
type ConfirmPaymentRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
	// ID of the Stripe Setup Intent to verify.
	//
	// Must be the Setup Intent most recently created for this session by Setup Registration Billing, and its status must be `succeeded`.
	SetupIntentID string `json:"setup_intent_id" validate:"required"`
}

var sampleConfirmPaymentRequest = &ConfirmPaymentRequest{
	SetupIntentID: "seti_1N4kLm2eZvKYlo2C0wFVpSbx",
}

func (*ConfirmPaymentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleConfirmPaymentRequest)
}

// Verifies that a Stripe Setup Intent succeeded and marks the registration session's payment as completed.
//
// A registration on a paid plan cannot be completed until this succeeds. Confirming a session whose payment is already recorded returns success without re-checking Stripe.
type ConfirmPaymentEndpoint struct{}

func (e *ConfirmPaymentEndpoint) Materialize() *apiendpoint.APIEndpoint[*ConfirmPaymentRequest, *apiresource.ConfirmPaymentResponse] {
	return (&apiendpoint.APIEndpoint[*ConfirmPaymentRequest, *apiresource.ConfirmPaymentResponse]{
		Title:             "Confirm Registration Payment",
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/actions/confirm-payment",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ConfirmPaymentRequest) (*apiresource.ConfirmPaymentResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).ConfirmPayment
		},
	})
}
