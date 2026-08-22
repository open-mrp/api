package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to verify a registration token.
type VerifyTokenRequest struct {
	// Verification token from the email link.
	Token string `json:"-" path:"token" validate:"required"`
}

// Verifies the token from the registration email, marking the session as email-verified and advancing it to the `user_details` step.
//
// A token is only accepted within 24 hours of the session's last update; Resend Verification Email issues a fresh one. Verifying a session that is already verified returns it unchanged.
type VerifyTokenEndpoint struct{}

func (e *VerifyTokenEndpoint) Materialize() *apiendpoint.APIEndpoint[*VerifyTokenRequest, *apiresource.RegistrationSession] {
	return (&apiendpoint.APIEndpoint[*VerifyTokenRequest, *apiresource.RegistrationSession]{
		Title: "Verify Registration Token",
		// #nosec G101 - API description, not a credential
		Method:            http.MethodPut,
		Route:             "/v1/auth/registration-sessions/{token}/actions/verify-token",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VerifyTokenRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).VerifyToken
		},
	})
}
