package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a user for a registration session.
type CreateUserRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
	// The user's display name.
	Name string `json:"name" validate:"required,max=255"`
	// Password for the new user.
	//
	// Must be 8–72 characters and contain at least one lowercase letter, one uppercase letter, one digit, and one special character.
	Password string `json:"password" validate:"required,password" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleCreateUserRequest = &CreateUserRequest{
	Name:     "Jane Smith",
	Password: "P@ssw0rd123!", // #nosec G101 - Sample data for API documentation
}

func (*CreateUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUserRequest)
}

// Creates the user for a registration session and signs the registrant in.
//
// The session's email must already be verified, and no user may exist for that email yet; someone who already has an account must sign in instead of registering again. On success the session advances to the `account_details` step and the response sets authentication cookies, so the remaining registration calls are made as the new user. Repeating the call on a session that already has a user re-issues cookies for that user instead of creating another.
type CreateUserEndpoint struct{}

func (e *CreateUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUserRequest, *apiresource.CreateUserResponse] {
	return (&apiendpoint.APIEndpoint[*CreateUserRequest, *apiresource.CreateUserResponse]{
		Title:             "Create User for Registration",
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/users",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateUserRequest) (*apiresource.CreateUserResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).CreateUser
		},
	})
}
