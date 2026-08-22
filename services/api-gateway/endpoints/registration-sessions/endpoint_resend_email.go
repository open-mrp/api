package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to resend the verification email.
type ResendEmailRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

// Resends the verification email for a registration session, generating a new token and invalidating the previous one.
//
// Rejected once the email has been verified or the registration has completed.
type ResendEmailEndpoint struct{}

func (e *ResendEmailEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResendEmailRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ResendEmailRequest, *apiresource.EmptyResource]{
		Title:             "Resend Verification Email",
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/actions/resend-verification-email",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ResendEmailRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).ResendVerificationEmail
		},
	})
}
