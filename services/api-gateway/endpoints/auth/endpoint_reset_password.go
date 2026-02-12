package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to reset a user's password
type ResetPasswordRequest struct {
	// The password reset token (from request_password_reset endpoint)
	Token string `json:"token" validate:"required"`
	// The new password of the user
	Password string `json:"password" validate:"required,password"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleResetPasswordRequest = &ResetPasswordRequest{
	Token:    apiresource.SampleAccessToken,
	Password: apiresource.SampleNewUserPassword,
}

func (*ResetPasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleResetPasswordRequest)
}

const resetPasswordEndpointDescription string = `This endpoint is used to reset a user's password using a password reset token.
Once completed, new access and refresh tokens are set in cookies, and previous tokens are revoked.`

type ResetPasswordEndpoint struct{}

func (e *ResetPasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResetPasswordRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*ResetPasswordRequest, *apiresource.EmptyResource]{
		Title:             "Reset Password",
		Description:       resetPasswordEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/passwords/actions/reset",
		ContentType:       "application/json",
		Request:           &ResetPasswordRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ResetPasswordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).ResetPassword
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
