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
	// The email address for the new user
	Email string `json:"email" validate:"required,custom_email"`
	// The password for the new user
	Password string `json:"password" validate:"required,password"` // #nosec G117 - Struct field, not a hardcoded credential
	// The full name of the new user
	Name string `json:"name" validate:"required"`
}

var sampleRegisterRequest = &RegisterRequest{
	Email:    apiresource.SampleUserEmail,
	Password: apiresource.SampleUserPassword,
	Name:     apiresource.SampleUserName,
}

func (*RegisterRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRegisterRequest)
}

const registerEndpointDescription string = `This endpoint is used to register a new user on the customer portal. Once completed, the user object is 
returned, and an access and refresh token are set in cookies.`

type RegisterEndpoint struct{}

func (e *RegisterEndpoint) Materialize() *apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User] {
	return &apiendpoint.APIEndpoint[*RegisterRequest, *apiresource.User]{
		Title:             "Register User",
		Description:       registerEndpointDescription,
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
			AllowUnknownJSONFields: false,
		},
	}
}
