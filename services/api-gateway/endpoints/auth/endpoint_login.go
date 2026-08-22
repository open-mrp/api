package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to log in a user.
type LoginRequest struct {
	// Username or email for authentication.
	Identifier string `json:"identifier" validate:"required,identifier"`
	// User password.
	Password string `json:"password" validate:"required,max=72" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleLoginRequest = &LoginRequest{
	Identifier: apiresource.SampleUserUsername,
	Password:   apiresource.SampleUserPassword,
}

func (*LoginRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleLoginRequest)
}

// Authenticates a user and returns the user object, setting access and refresh tokens in cookies.
//
// Failed attempts are throttled per identifier: after 10 failures within 5 minutes, further attempts for that identifier are rejected with a rate-limit error until the window passes. Invalid credentials always return the same generic error, whether or not the identifier exists.
type LoginEndpoint struct{}

func (e *LoginEndpoint) Materialize() *apiendpoint.APIEndpoint[*LoginRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*LoginRequest, *apiresource.User]{
		Title:             "Login User",
		Method:            http.MethodPost,
		Route:             "/v1/auth/actions/login",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ObjectType:        constants.ObjectTypeUser,
		ServiceHandler: func(svc any) func(ctx context.Context, req *LoginRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(AuthSvc).Login
		},
	})
}
