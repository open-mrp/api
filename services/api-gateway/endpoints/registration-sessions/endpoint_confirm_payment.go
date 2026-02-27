package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ConfirmPaymentRequest is the request to confirm payment for a registration session.
type ConfirmPaymentRequest struct {
	// The session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
	// The Stripe checkout session ID to verify.
	CheckoutSessionID string `json:"checkout_session_id" validate:"required"`
}

var sampleConfirmPaymentRequest = &ConfirmPaymentRequest{
	CheckoutSessionID: "cs_test_a1VnbGQ4ZTFRdGRqUWpYR3h6OG",
}

func (*ConfirmPaymentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleConfirmPaymentRequest)
}

const confirmPaymentEndpointDescription string = `Confirms that a Stripe checkout session payment has completed for the given registration session.
Retrieves the checkout status from Stripe and, if complete, marks the registration session's payment as done.`

type ConfirmPaymentEndpoint struct{}

func (e *ConfirmPaymentEndpoint) Materialize() *apiendpoint.APIEndpoint[*ConfirmPaymentRequest, *apiresource.ConfirmPaymentResponse] {
	return &apiendpoint.APIEndpoint[*ConfirmPaymentRequest, *apiresource.ConfirmPaymentResponse]{
		Title:             "Confirm Registration Payment",
		Description:       confirmPaymentEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/actions/confirm-payment",
		ContentType:       "application/json",
		Request:           &ConfirmPaymentRequest{},
		Response:          &apiresource.ConfirmPaymentResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ConfirmPaymentRequest) (*apiresource.ConfirmPaymentResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).ConfirmPayment
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
