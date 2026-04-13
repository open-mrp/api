package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to resend the verification email.
type ResendEmailRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

type ResendEmailEndpoint struct{}

func (e *ResendEmailEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResendEmailRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*ResendEmailRequest, *apiresource.EmptyResource]{
		Title:             "Resend Verification Email",
		Description:       "Resends the verification email for a registration session, generating a new token and invalidating the previous one.",
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/actions/resend-verification-email",
		ContentType:       "application/json",
		Request:           &ResendEmailRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ResendEmailRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).ResendVerificationEmail
		},
	}
}
