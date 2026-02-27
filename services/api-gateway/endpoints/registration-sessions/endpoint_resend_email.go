package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to resend the verification email
type ResendEmailRequest struct {
	// The session ID (from path).
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

const resendEmailDescription string = `Resends the verification email for an existing registration session. A new verification
token is generated and the previous token is invalidated.`

type ResendEmailEndpoint struct{}

func (e *ResendEmailEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResendEmailRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*ResendEmailRequest, *apiresource.EmptyResource]{
		Title:             "Resend Verification Email",
		Description:       resendEmailDescription,
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
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
