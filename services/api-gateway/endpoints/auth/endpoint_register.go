package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to register a new user
type RegisterRequest struct {
	// The email address for the new user.
	Email string `json:"email" validate:"required,custom_email"`
	// The password for the new user.
	Password string `json:"password" validate:"required,password"` // #nosec G117 - Struct field, not a hardcoded credential
	// The full name of the new user.
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

type RegisterEndpoint struct{}

func (e *RegisterEndpoint) Materialize() *apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User] {
	return &apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User]{
		Title:             "Register User",
		Description:       "Registers a new user on the customer portal, returning the user object and setting access and refresh tokens in cookies.",
		Method:            http.MethodPost,
		Route:             "/v1/auth/users",
		ContentType:       "application/json",
		Request:           &RegisterRequest{},
		Response:          &apiresource.User{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RegisterRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(AuthSvc).Register
		},
		Extras: apiendpoint.APIEndpointExtras{
			ShieldRequestBody: true,
		},
	}
}
