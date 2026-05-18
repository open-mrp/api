package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
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
type LoginEndpoint struct{}

func (e *LoginEndpoint) Materialize() *apiendpoint.APIEndpoint[*LoginRequest, *apiresource.User] {
	return (&apiendpoint.APIEndpoint[*LoginRequest, *apiresource.User]{
		Title:             "Login User",
		Method:            http.MethodPost,
		Route:             "/v1/auth/actions/login",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *LoginRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(AuthSvc).Login
		},
	})
}
