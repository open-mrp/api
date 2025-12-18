package authep

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/validate"
)

// The request to reset a user's password
type ResetPasswordRequest struct {
	// The password reset token
	Token string `json:"token" validate:"required"`
	// The new password of the user
	Password string `json:"password" validate:"required"`
}

func (rr *ResetPasswordRequest) Validate() error {
	v := validate.New()

	if rr.Token == "" {
		v.AddError("token", "Token is required.")
	}
	validate.ValidatePasswordPlaintext(v, rr.Password)

	if !v.Valid() {
		var errorMessages []string
		for field, message := range v.Errors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", field, message))
		}
		return contracts.NewValidationError(strings.Join(errorMessages, "; "))
	}

	return nil
}

var sampleResetPasswordRequest = &ResetPasswordRequest{
	Token:    apiresource.SampleAccessToken,
	Password: apiresource.SampleNewUserPassword,
}

func (*ResetPasswordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleResetPasswordRequest)
}

const resetPasswordEndpointDescription = `This endpoint is utilized to reset a user's password using a password reset token.
Once completed, new access and refresh tokens are set in cookies. Learn more about authentication and authorization in 
our [documentation](https://docs.augno.com/guides/authentication).
`

type ResetPasswordEndpoint struct {
	apiendpoint.APIEndpoint[*ResetPasswordRequest, *apiresource.EmptyResource]

	group    *apiendpoint.APIEndpointGroup
	service  AuthCtrl
	platform constants.PlatformMode
	bindOnce sync.Once
	handler  http.HandlerFunc
}

func (e *ResetPasswordEndpoint) Materialize() apiendpoint.APIEndpointer {
	e.APIEndpoint = apiendpoint.APIEndpoint[*ResetPasswordRequest, *apiresource.EmptyResource]{
		Title:             "Reset Password",
		Description:       resetPasswordEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/passwords/actions/reset",
		ContentType:       "application/json",
		Request:           &ResetPasswordRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		IsPublic:          true,
		Handler: func(ctrl any) apiendpoint.HandlerFunc[
			*ResetPasswordRequest, *apiresource.EmptyResource,
		] {
			return apiendpoint.HandlerFunc[
				*ResetPasswordRequest, *apiresource.EmptyResource,
			](func(ctx context.Context, req *ResetPasswordRequest) (*apiresource.EmptyResource, *contracts.APIError) {
				return ctrl.(AuthCtrl).ResetPassword(ctx, req)
			})
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
	return e
}

func (e *ResetPasswordEndpoint) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		be := apiendpoint.Bind(e.APIEndpoint, e.service)
		e.handler = httptransport.ConvertToHTTPHandler(be)
	})
	return e.handler
}

func (e *ResetPasswordEndpoint) WithGroup(g *apiendpoint.APIEndpointGroup, service AuthCtrl, platform constants.PlatformMode) *ResetPasswordEndpoint {
	e.group = g
	e.service = service
	e.platform = platform
	return e
}
