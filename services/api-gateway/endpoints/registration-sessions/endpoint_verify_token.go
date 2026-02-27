package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to verify a registration token
type VerifyTokenRequest struct {
	// The verification token from the email link.
	Token string `json:"-" path:"token" validate:"required"`
}

const verifyTokenEndpointDescription string = `Verifies the email token sent during registration. Marks the session's email as verified and
advances the registration flow to the next step. Idempotent: repeated calls with the same token return the same session.` // #nosec G101 - API description, not a credential

type VerifyTokenEndpoint struct{}

func (e *VerifyTokenEndpoint) Materialize() *apiendpoint.APIEndpoint[*VerifyTokenRequest, *apiresource.RegistrationSession] {
	return &apiendpoint.APIEndpoint[*VerifyTokenRequest, *apiresource.RegistrationSession]{
		Title:             "Verify Registration Token",
		Description:       verifyTokenEndpointDescription,
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
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
