package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to create a user for a registration session
type CreateUserRequest struct {
	// The session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
	// Display name for the new user.
	Name string `json:"name" validate:"required"`
	// Password for the new user account.
	Password string `json:"password" validate:"required,password"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleCreateUserRequest = &CreateUserRequest{
	Name:     "Jane Smith",
	Password: "P@ssw0rd123!", // #nosec G101 - Sample data for API documentation
}

func (*CreateUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUserRequest)
}

const createUserDescription string = `Creates a user for the given registration session. If the session email matches an existing user,
that user is associated with the session. Otherwise a new user is created with the provided name and password.
Returns the user ID. Idempotent: if a user is already associated with the session, returns the existing user ID.`

type CreateUserEndpoint struct{}

func (e *CreateUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUserRequest, *apiresource.CreateUserResponse] {
	return &apiendpoint.APIEndpoint[*CreateUserRequest, *apiresource.CreateUserResponse]{
		Title:             "Create User for Registration",
		Description:       createUserDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/users",
		ContentType:       "application/json",
		Request:           &CreateUserRequest{},
		Response:          &apiresource.CreateUserResponse{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateUserRequest) (*apiresource.CreateUserResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).CreateUser
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
			ShieldRequestBody:      true,
		},
	}
}
