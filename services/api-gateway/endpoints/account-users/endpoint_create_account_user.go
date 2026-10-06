package accountuserep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
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
	// Whether the user can be assigned as a sales representative on orders, territories, and targets.
	//
	// Defaults to false. Forced true for the `sales_rep` role type and rejected for scanner and agent roles.
	IsCommissionEligible field.Optional[bool] `json:"is_commission_eligible,omitzero"`
	// Notification preference toggles for the new user.
	//
	// Only applies when adding a user to a customer or supplier account you manage; ignored when adding a user to your own account. The enabled types are returned in the user's `notification_types`. When the user is one you previously removed, the toggles apply over the preferences they had before removal, and types you leave out keep their previous state.
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
// If no user with the given email or username exists, a new user is created; a user created with an email address is sent a welcome email containing a generated password and a link to sign in, unless they are being added to a supplier account, since suppliers have no portal to sign in to. A user added to a customer account signs in through your customer portal, on its verified custom domain when you have one. If a matching user already exists, that user is added to the account instead, and a user you previously removed is restored rather than duplicated. Adding a user to your own account consumes a seat and is rejected once your plan's seat limit is reached.
//
// You can add users to a customer or supplier account only while you manage it alone: an account that runs its own OpenMRP subscription, or that another account also has as a customer or supplier, manages its own users, and the request is refused.
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
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTeamUsers, Action: types.ActionCreate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionCreate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionCreate}},
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
