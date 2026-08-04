package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to reset a user's password.
type ResetPasswordRequest struct {
	// Password reset token taken from the `t` query parameter of the link in the password reset email.
	//
	// The token expires 15 minutes after the email is sent; after that the user has to request a new reset email.
	Token string `json:"token" validate:"required" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
	// New password to set for the user.
	Password string `json:"password" validate:"required,password" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleResetPasswordRequest = &ResetPasswordRequest{
	Token:    apiresource.SampleAccessToken,
	Password: apiresource.SampleNewUserPassword,
}

func (*ResetPasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleResetPasswordRequest)
}

// Sets a new password using a password reset token and signs the user in.
//
// All of the user's existing refresh tokens are revoked, signing out their other sessions, and fresh access and refresh tokens are set in cookies. A confirmation email is sent to the user.
type ResetPasswordEndpoint struct{}

func (e *ResetPasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResetPasswordRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ResetPasswordRequest, *apiresource.EmptyResource]{
		Title:             "Reset Password",
		Method:            http.MethodPost,
		Route:             "/v1/auth/passwords/actions/reset",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ResetPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).ResetPassword
		},
	})
}
