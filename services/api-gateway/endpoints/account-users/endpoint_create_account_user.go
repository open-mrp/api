package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create an account user.
type CreateAccountUserRequest struct {
	// User display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// User email address.
	//
	// Either `email` or `username` must be provided. If a user with this email already exists, that user is added to the account instead of a new user being created, and the request fails with a conflict if they are already an active member of it.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// Unique username.
	//
	// 3–255 characters; letters, numbers, underscores, and hyphens. Either `email` or `username` must be provided. Providing a username without an email creates a scanning station user.
	Username field.Optional[string] `json:"username,omitzero" validate:"omitempty,username"`
	// Password for scanning station users.
	//
	// Required when creating a scanning station user (username without email) and rejected for all other users, who instead receive a generated password in their welcome email. Must be 8–72 characters and include an uppercase letter, a lowercase letter, a number, and a special character.
	Password field.Optional[string] `json:"password,omitzero" validate:"omitempty,password" sensitive:"true"` // #nosec G117 -- API request field for user password input
	// ID of the role to assign to the user.
	//
	// The role you supply can be overridden: users added to a customer account always receive the shared customer role so their portal capabilities stay permission-driven, and scanning station users in any other account receive the scanner role. Supplying a role whose type is `sales_rep` normalizes to the account's canonical sales-rep role.
	RoleID field.Optional[string] `json:"role_id,omitzero"`
	// ID of the department to assign to the user.
	//
	// The department must already exist in the account you are acting in.
	DepartmentID field.Optional[string] `json:"department_id,omitzero"`
	// Notification preference toggles for the new user.
	//
	// Only applies when creating a user in another account you manage (cross-account); ignored when creating a user in your own account.
	Preferences []NotificationPreferenceItem `json:"preferences,omitzero"`
}

var sampleCreateAccountUserName = apiresource.SampleUserName
var sampleCreateAccountUserEmail = apiresource.SampleUserEmail
var sampleCreateAccountUserUsername = apiresource.SampleUserUsername
var sampleCreateAccountUserPassword = apiresource.SampleUserPassword
var sampleCreateAccountUserRoleID = apiresource.SampleRoleID
var sampleCreateAccountUserDepartmentID = apiresource.SampleDepartmentID
var sampleCreateAccountUserRequest = &CreateAccountUserRequest{
	Name:         field.Some(sampleCreateAccountUserName),
	Email:        field.Some(sampleCreateAccountUserEmail),
	Username:     field.Some(sampleCreateAccountUserUsername),
	Password:     field.Some(sampleCreateAccountUserPassword),
	RoleID:       field.SomePtr(&sampleCreateAccountUserRoleID),
	DepartmentID: field.SomePtr(&sampleCreateAccountUserDepartmentID),
	Preferences: []NotificationPreferenceItem{
		{NotificationTypeCode: constants.AccountRelationNotificationTypeOrderAcknowledgement, Enabled: true},
	},
}

func (*CreateAccountUserRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountUserRequest)
}

// Adds a user to the account you are acting in.
//
// If no user with the given email or username exists, a new user is created; a user created with an email address is sent a welcome email containing a generated password, unless they are being added to a supplier account, since suppliers have no portal to sign in to. If a matching user already exists, that user is added to the account instead, and a user you previously removed is restored rather than duplicated. Adding a user to your own account consumes a seat and is rejected once your plan's seat limit is reached.
//
// When you add a user to a customer or supplier account that has its own Augno subscription, the membership is created disabled and has to be activated before that user can sign in.
type CreateAccountUserEndpoint struct{}

func (e *CreateAccountUserEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountUserRequest, *apiresource.AccountUser] {
	return (&apiendpoint.APIEndpoint[*CreateAccountUserRequest, *apiresource.AccountUser]{
		Title:               "Create Account User",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/identity/account-users",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionCreate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
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
