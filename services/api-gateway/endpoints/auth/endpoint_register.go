package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
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
	AccountSlug field.Optional[string] `json:"account_slug,omitzero" validate:"omitempty"`
}

var sampleRegisterRequest = &RegisterRequest{
	Email:    apiresource.SampleUserEmail,
	Password: apiresource.SampleUserPassword,
	Name:     apiresource.SampleUserName,
}

func (*RegisterRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRegisterRequest)
}

// Registers a user on the customer portal.
//
// Returns the new user object and sets access and refresh tokens in cookies. If the email is already registered, the request fails with a generic validation error (so existing emails are not revealed) and an "already registered" email containing a magic login link is sent to the existing user instead.
type RegisterEndpoint struct{}

func (e *RegisterEndpoint) Materialize() *apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User]{
		Title:             "Register User",
		Method:            http.MethodPost,
		Route:             "/v1/auth/users",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ObjectType:        constants.ObjectTypeUser,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RegisterRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(AuthSvc).Register
		},
	})
}
