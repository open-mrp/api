package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to login a user
type LoginRequest struct {
	// The username or email for authentication.
	Identifier string `json:"identifier" validate:"required,identifier"`
	// The password of the user.
	Password string `json:"password" validate:"required,max=72"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleLoginRequest = &LoginRequest{
	Identifier: apiresource.SampleUserUsername,
	Password:   apiresource.SampleUserPassword,
}

func (*LoginRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleLoginRequest)
}

type LoginEndpoint struct{}

func (e *LoginEndpoint) Materialize() *apiendpoint.APIEndpoint[*LoginRequest, *apiresource.User] {
	return &apiendpoint.APIEndpoint[*LoginRequest, *apiresource.User]{
		Title:             "Login User",
		Description:       "Authenticates a user and returns the user object, setting access and refresh tokens in cookies.",
		Method:            http.MethodPost,
		Route:             "/v1/auth/actions/login",
		ContentType:       "application/json",
		Request:           &LoginRequest{},
		Response:          &apiresource.User{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *LoginRequest) (*apiresource.User, *apierror.APIError) {
			return svc.(AuthSvc).Login
		},
		Extras: apiendpoint.APIEndpointExtras{
			ShieldRequestBody: true,
		},
	}
}
