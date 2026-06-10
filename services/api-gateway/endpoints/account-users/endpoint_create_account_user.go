package accountuserep

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

// Request to create an account user.
type CreateAccountUserRequest struct {
	// User display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// User email address.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// Unique username (3–255 chars; letters, numbers, underscores, hyphens).
	Username field.Optional[string] `json:"username,omitzero" validate:"omitempty,username"`
	// Password. Only used for scanner-role users (scanning stations).
	// Must be 8–72 chars and include upper, lower, number, and special character.
	Password field.Optional[string] `json:"password,omitzero" validate:"omitempty,password" sensitive:"true"` // #nosec G117 -- API request field for user password input
	// Role assigned to the user.
	RoleID field.Optional[string] `json:"role_id,omitzero"`
	// Department assigned to the user.
	DepartmentID field.Optional[string] `json:"department_id,omitzero"`
	// Notification preferences for the user (external targets only).
	Preferences []NotificationPreferenceItem `json:"preferences,omitzero"`
}

var sampleCreateAccountUserName = apiresource.SampleUserName
var sampleCreateAccountUserEmail = apiresource.SampleUserEmail
var sampleCreateAccountUserUsername = apiresource.SampleUserUsername
var sampleCreateAccountUserPassword = apiresource.SampleUserPassword
var sampleCreateAccountUserRoleID = apiresource.SampleRoleID
var sampleCreateAccountUserRequest = &CreateAccountUserRequest{
	Name:     field.Some(sampleCreateAccountUserName),
	Email:    field.Some(sampleCreateAccountUserEmail),
	Username: field.Some(sampleCreateAccountUserUsername),
	Password: field.Some(sampleCreateAccountUserPassword),
	RoleID:   field.SomePtr(&sampleCreateAccountUserRoleID),
	Preferences: []NotificationPreferenceItem{
		{NotificationTypeCode: constants.AccountRelationNotificationTypeOrderAcknowledgement, Enabled: true},
	},
}

func (*CreateAccountUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountUserRequest)
}

// Creates a new account user and invites them to the target account.
type CreateAccountUserEndpoint struct{}

func (e *CreateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountUserRequest, *apiresource.AccountUser] {
	return (&apiendpoint.APIEndpoint[*CreateAccountUserRequest, *apiresource.AccountUser]{
		Title:             "Create Account User",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/account-users",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
			return svc.(AccountUserSvc).CreateAccountUser
		},
		LocationFunc: func(resp *apiresource.AccountUser) string {
			return "/v1/identity/account-users/" + resp.ID
		},
		ObjectType: constants.ObjectTypeAccountUser,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"user", "role", "department"},
		}),
	})
}
