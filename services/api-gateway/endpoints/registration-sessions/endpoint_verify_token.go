package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to verify a registration token.
type VerifyTokenRequest struct {
	// Verification token from the email link.
	Token string `json:"-" path:"token" validate:"required"`
}

// Verifies the email token sent during registration, marking the session as email-verified and advancing to the next step.
type VerifyTokenEndpoint struct{}

func (e *VerifyTokenEndpoint) Materialize() *apiendpoint.APIEndpoint[*VerifyTokenRequest, *apiresource.RegistrationSession] {
	return (&apiendpoint.APIEndpoint[*VerifyTokenRequest, *apiresource.RegistrationSession]{
		Title: "Verify Registration Token",
		// #nosec G101 - API description, not a credential
		Method:            http.MethodPut,
		Route:             "/v1/auth/registration-sessions/{token}/actions/verify-token",
		ContentType:       "application/json",
		Request:           &VerifyTokenRequest{},
		Response:          &apiresource.RegistrationSession{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *VerifyTokenRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).VerifyToken
		},
	}).WithDocSource(e)
}
