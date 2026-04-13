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
	// Password reset token.
	Token string `json:"token" validate:"required"`
	// New password.
	Password string `json:"password" validate:"required,password"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleResetPasswordRequest = &ResetPasswordRequest{
	Token:    apiresource.SampleAccessToken,
	Password: apiresource.SampleNewUserPassword,
}

func (*ResetPasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleResetPasswordRequest)
}

type ResetPasswordEndpoint struct{}

func (e *ResetPasswordEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResetPasswordRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*ResetPasswordRequest, *apiresource.EmptyResource]{
		Title:             "Reset Password",
		Description:       "Resets a user's password using a password reset token, revoking previous tokens and setting new ones in cookies.",
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
			ShieldRequestBody: true,
		},
	}
}
