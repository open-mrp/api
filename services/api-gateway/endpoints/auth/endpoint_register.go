package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to register a user.
type RegisterRequest struct {
	// Email address the user will sign in with.
	//
	// Must not already belong to a user; the request is rejected without revealing that the address is taken.
	Email string `json:"email" validate:"required,custom_email"`
	// Password the user will sign in with.
	Password string `json:"password" validate:"required,password" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
	// Full name of the user, used to address them in emails.
	Name string `json:"name" validate:"required"`
	// Slug of the customer portal the user is registering from.
	//
	// Only affects the "already registered" email sent when the address is taken: it points the magic-login link back at that portal instead of the generic dashboard. Accounts with a verified custom portal domain use that domain in the link instead of the slug.
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
// Returns the new user object, sets access and refresh tokens in cookies, and sends the user a welcome email. Registering creates the user record only; membership in an account is granted separately.
//
// If the email is already registered, the request fails with a generic validation error (so existing emails are not revealed) and an "already registered" email containing a magic login link is sent to the existing user instead.
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
