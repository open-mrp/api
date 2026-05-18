package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to register a user.
type RegisterRequest struct {
	// Email address.
	Email string `json:"email" validate:"required,custom_email"`
	// User password.
	Password string `json:"password" validate:"required,password" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
	// Full name.
	Name string `json:"name" validate:"required"`
	// When registering from a customer portal, scopes the magic-login link in the "already registered" email.
	AccountSlug *string `json:"account_slug,omitempty" validate:"omitempty"`
}

var sampleRegisterRequest = &RegisterRequest{
	Email:    apiresource.SampleUserEmail,
	Password: apiresource.SampleUserPassword,
	Name:     apiresource.SampleUserName,
}

func (*RegisterRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRegisterRequest)
}

// Registers a user on the customer portal. Returns the user object and sets access and refresh tokens in cookies.
type RegisterEndpoint struct{}

func (e *RegisterEndpoint) Materialize() *apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User]{
		Title:             "Register User",
		Method:            http.MethodPost,
		Route:             "/v1/auth/users",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RegisterRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(AuthSvc).Register
		},
	})
}
