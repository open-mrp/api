package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create an account user.
type CreateAccountUserRequest struct {
	// User display name.
	Name *string `json:"name" validate:"omitempty,max=255"`
	// User email address.
	Email *string `json:"email" validate:"omitnil,custom_email,max=255"`
	// Unique username (3–255 chars; letters, numbers, underscores, hyphens).
	Username *string `json:"username" validate:"omitempty,username"`
	// Password. Only used for scanner-role users (scanning stations).
	// Must be 8–72 chars and include upper, lower, number, and special character.
	Password *string `json:"password" validate:"omitempty,password" sensitive:"true"` // #nosec G117 -- API request field for user password input
	// Role assigned to the user.
	RoleID *string `json:"role_id,omitempty" validate:"omitempty"`
	// Department assigned to the user.
	DepartmentID *string `json:"department_id,omitempty" validate:"omitempty"`
	// Notification preferences for the user (external targets only).
	Preferences []NotificationPreferenceItem `json:"preferences,omitempty"`
}

var sampleCreateAccountUserName = apiresource.SampleUserName
var sampleCreateAccountUserEmail = apiresource.SampleUserEmail
var sampleCreateAccountUserUsername = apiresource.SampleUserUsername
var sampleCreateAccountUserPassword = apiresource.SampleUserPassword
var sampleCreateAccountUserRoleID = apiresource.SampleRoleID
var sampleCreateAccountUserRequest = &CreateAccountUserRequest{
	Name:     &sampleCreateAccountUserName,
	Email:    &sampleCreateAccountUserEmail,
	Username: &sampleCreateAccountUserUsername,
	Password: &sampleCreateAccountUserPassword,
	RoleID:   &sampleCreateAccountUserRoleID,
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
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAccountUser,
			Fields:     []string{"role", "department"},
		}),
	})
}
